/*
 * gobot -- a useful discord bot
 *
 * Copyright (C) 2024 Sabafly Developers
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package tui

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"

	"github.com/sabafly/gobot/internal/adminclient"
)

const defaultAdminURL = "http://127.0.0.1:8080"

type cliFlags struct {
	URL         string
	SessionFile string
	NoBrowser   bool
}

// Command creates the `gobot tui` Cobra command tree (`tui`, `tui login`, `tui logout`, `tui status`).
func Command() *cobra.Command {
	defURL := strings.TrimSpace(os.Getenv("GOBOT_ADMIN_URL"))
	if defURL == "" {
		defURL = defaultAdminURL
	}
	flags := &cliFlags{}

	cmd := &cobra.Command{
		Use:   "tui",
		Short: "Launch the gobot management TUI dashboard",
		RunE: func(c *cobra.Command, args []string) error {
			store, client, baseURL, err := initClientAndStore(flags)
			if err != nil {
				return err
			}
			app := NewApp(AppOptions{
				BaseURL:   baseURL,
				Client:    client,
				Store:     store,
				NoBrowser: flags.NoBrowser,
			})
			defer app.Close()

			_, err = tea.NewProgram(app).Run()
			return err
		},
	}

	cmd.PersistentFlags().StringVarP(&flags.URL, "url", "u", defURL, "Admin API base URL, including base_path if configured (env: GOBOT_ADMIN_URL)")
	cmd.PersistentFlags().StringVar(&flags.SessionFile, "session-file", "", "Path to local session JSON file (default: user config dir)")
	cmd.PersistentFlags().BoolVar(&flags.NoBrowser, "no-browser", false, "Do not attempt to automatically open a browser for OAuth2")

	cmd.AddCommand(
		&cobra.Command{
			Use:   "login",
			Short: "Authenticate with gobot Admin API via Discord OAuth2",
			RunE: func(c *cobra.Command, args []string) error {
				store, client, baseURL, err := initClientAndStore(flags)
				if err != nil {
					return err
				}
				return runLogin(c.Context(), c.OutOrStdout(), client, store, baseURL, flags.NoBrowser, loginPollInterval)
			},
		},
		&cobra.Command{
			Use:   "logout",
			Short: "Revoke the stored admin session on the server and remove it locally",
			RunE: func(c *cobra.Command, args []string) error {
				store, client, baseURL, err := initClientAndStore(flags)
				if err != nil {
					return err
				}
				return runLogout(c.Context(), c.OutOrStdout(), client, store, baseURL)
			},
		},
		&cobra.Command{
			Use:   "status",
			Short: "Print current admin API connection and operator session info",
			RunE: func(c *cobra.Command, args []string) error {
				store, client, baseURL, err := initClientAndStore(flags)
				if err != nil {
					return err
				}
				return runStatus(c.Context(), c.OutOrStdout(), client, store, baseURL)
			},
		},
	)

	return cmd
}

func initClientAndStore(flags *cliFlags) (*adminclient.FileSessionStore, *adminclient.Client, string, error) {
	baseURL := adminclient.NormalizeBaseURL(flags.URL)
	if baseURL == "" {
		return nil, nil, "", errors.New("tui: admin URL cannot be empty")
	}
	store, err := adminclient.NewFileSessionStore(flags.SessionFile)
	if err != nil {
		return nil, nil, "", err
	}
	var token string
	if sess, ok, gErr := store.Get(baseURL); gErr == nil && ok {
		token = sess.Token
	}
	client := adminclient.New(baseURL, token)
	return store, client, baseURL, nil
}

func runLogin(ctx context.Context, out io.Writer, client *adminclient.Client, store *adminclient.FileSessionStore, baseURL string, noBrowser bool, interval time.Duration) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if interval <= 0 {
		interval = loginPollInterval
	}
	startResp, err := client.StartDiscordLogin(ctx, tuiClientInfo())
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(out, "Authorize URL:\n%s\n", startResp.AuthorizeURL)
	if !noBrowser {
		if bErr := openBrowserFn(startResp.AuthorizeURL); bErr != nil {
			_, _ = fmt.Fprintf(out, "Note: could not launch browser automatically (%v)\n", bErr)
		}
	}
	_, _ = fmt.Fprintln(out, "Waiting for Discord OAuth2 approval...")

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		if !startResp.ExpiresAt.IsZero() && time.Now().After(startResp.ExpiresAt) {
			return errors.New("login request expired")
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}

		pollResp, pErr := client.PollDiscordLogin(ctx, startResp.LoginID, startResp.PollSecret)
		if pErr != nil {
			if strings.Contains(strings.ToLower(pErr.Error()), "404") {
				return pErr
			}
			continue
		}
		switch pollResp.Status {
		case "pending":
			continue
		case "denied", "expired":
			if pollResp.Error != "" {
				return fmt.Errorf("login %s: %s", pollResp.Status, pollResp.Error)
			}
			return fmt.Errorf("login %s", pollResp.Status)
		case "approved":
			client.Token = pollResp.SessionToken
			var username, displayName, userID, role string
			if pollResp.Principal != nil {
				username = pollResp.Principal.Username
				displayName = pollResp.Principal.DisplayName
				userID = pollResp.Principal.UserID
				role = pollResp.Principal.Role
			}
			if store != nil {
				exp := time.Now().UTC().Add(7 * 24 * time.Hour)
				if pollResp.ExpiresAt != nil {
					exp = *pollResp.ExpiresAt
				}
				if sErr := store.Save(baseURL, adminclient.StoredSession{
					BaseURL:     baseURL,
					Token:       pollResp.SessionToken,
					UserID:      userID,
					Username:    username,
					DisplayName: displayName,
					Role:        role,
					ExpiresAt:   exp,
					SavedAt:     time.Now().UTC(),
				}); sErr != nil {
					return sErr
				}
			}
			_, _ = fmt.Fprintf(out, "Logged in to %s as @%s [%s]\n", baseURL, username, role)
			return nil
		}
	}
}

func runLogout(ctx context.Context, out io.Writer, client *adminclient.Client, store *adminclient.FileSessionStore, baseURL string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if strings.TrimSpace(client.Token) != "" {
		if err := client.Logout(ctx); err != nil && !adminclient.IsUnauthorized(err) {
			_, _ = fmt.Fprintf(out, "Warning: server logout returned %v\n", err)
		}
	}
	if store != nil {
		if err := store.Delete(baseURL); err != nil {
			return err
		}
	}
	client.Token = ""
	_, _ = fmt.Fprintf(out, "Logged out from %s\n", baseURL)
	return nil
}

func runStatus(ctx context.Context, out io.Writer, client *adminclient.Client, store *adminclient.FileSessionStore, baseURL string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if strings.TrimSpace(client.Token) == "" {
		_, _ = fmt.Fprintf(out, "Server: %s\nStatus: Not logged in\n", baseURL)
		return nil
	}
	me, err := client.Me(ctx)
	if err != nil {
		if store != nil && adminclient.IsUnauthorized(err) {
			_ = store.Delete(baseURL)
		}
		return err
	}
	_, _ = fmt.Fprintf(out, "Server: %s\nStatus: Authenticated\nUser: @%s (%s / %s)\nRole: %s\n",
		baseURL, me.Username, me.DisplayName, me.UserID, me.Role)
	return nil
}
