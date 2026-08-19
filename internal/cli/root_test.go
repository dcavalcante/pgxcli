package cli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/balajz/pgxcli/internal/cliio"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type dbAndUserTestCase struct {
	dbnameOpt string
	userOpt   string
	argDB     string
	argUser   string

	expectedDB   string
	expectedUser string
}

type testApp struct {
	closeErr    error
	closeCalled bool
}

func (a *testApp) Start(context.Context) error { return nil }

func (a *testApp) Close() error {
	a.closeCalled = true
	return a.closeErr
}

func TestCloseResourcesClosesEverythingAndJoinsErrors(t *testing.T) {
	t.Parallel()

	appErr := errors.New("app close failed")
	loggerErr := errors.New("logger close failed")
	var calls []string

	err := closeResources(
		func() error {
			calls = append(calls, "app")
			return appErr
		},
		func() error {
			calls = append(calls, "client")
			return nil
		},
		func() error {
			calls = append(calls, "logger")
			return loggerErr
		},
	)

	assert.Equal(t, []string{"app", "client", "logger"}, calls)
	assert.ErrorIs(t, err, appErr)
	assert.ErrorIs(t, err, loggerErr)
}

func TestPersistentPostRunCleanupAndFarewell(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		childCommand bool
		cleanupFails bool
		wantFarewell bool
	}{
		{name: "root success", wantFarewell: true},
		{name: "root failure", cleanupFails: true},
		{name: "child success", childCommand: true},
		{name: "child failure", childCommand: true, cleanupFails: true},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			var output bytes.Buffer
			var closeErr error
			if testCase.cleanupFails {
				closeErr = errors.New("history save failed")
			}
			testApplication := &testApp{closeErr: closeErr}
			cliCtx := &CliContext{
				App:     testApplication,
				Printer: cliio.NewPgxPrinter(&output, io.Discard),
			}
			rootCmd := NewRootCmd(context.Background(), cliCtx)
			cmd := rootCmd
			if testCase.childCommand {
				cmd = &cobra.Command{Use: "export"}
				rootCmd.AddCommand(cmd)
			}

			err := rootCmd.PersistentPostRunE(cmd, nil)
			if closeErr != nil {
				require.ErrorIs(t, err, closeErr)
			} else {
				require.NoError(t, err)
			}
			assert.True(t, testApplication.closeCalled)
			assert.Equal(t, testCase.wantFarewell, strings.Contains(output.String(), "Thanks for using Pgxcli."))
		})
	}
}

func TestPromptPasswordFallsBackToFullLineInput(t *testing.T) {
	oldStdin := os.Stdin
	stdin, writer, err := os.Pipe()
	require.NoError(t, err)
	t.Cleanup(func() {
		os.Stdin = oldStdin
		_ = stdin.Close()
	})

	os.Stdin = stdin
	_, err = writer.WriteString("correct horse battery staple\r\n")
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	got, err := promptPassword("Enter password")

	require.NoError(t, err)
	assert.Equal(t, "correct horse battery staple", got)
}

func TestResolveDBAndUser(t *testing.T) {
	testcases := []struct {
		name  string
		input dbAndUserTestCase
	}{
		{
			name: "Both flags provided, args ignored",
			input: dbAndUserTestCase{
				dbnameOpt:    "flagDB",
				userOpt:      "flagUser",
				argDB:        "argDB",
				argUser:      "argUser",
				expectedDB:   "flagDB",
				expectedUser: "flagUser",
			},
		},
		{
			name: "Only dbname flag provided",
			input: dbAndUserTestCase{
				dbnameOpt:    "flagDB",
				userOpt:      "",
				argDB:        "argDB",
				argUser:      "argUser",
				expectedDB:   "flagDB",
				expectedUser: "argUser",
			},
		},
		{
			name: "Only username flag provided",
			input: dbAndUserTestCase{
				dbnameOpt:    "",
				userOpt:      "flagUser",
				argDB:        "argDB",
				argUser:      "argUser",
				expectedDB:   "argDB",
				expectedUser: "flagUser",
			},
		},
		{
			name: "No flags provided, args used",
			input: dbAndUserTestCase{
				dbnameOpt:    "",
				userOpt:      "",
				argDB:        "argDB",
				argUser:      "argUser",
				expectedDB:   "argDB",
				expectedUser: "argUser",
			},
		},
		{
			name: "No flags or args provided",
			input: dbAndUserTestCase{
				dbnameOpt:    "",
				userOpt:      "",
				argDB:        "",
				argUser:      "",
				expectedDB:   "",
				expectedUser: "",
			},
		},
		{
			name: "Only dbname flag and argDB provided",
			input: dbAndUserTestCase{
				dbnameOpt:    "flagDB",
				userOpt:      "",
				argDB:        "argDB",
				argUser:      "",
				expectedDB:   "flagDB",
				expectedUser: "argDB",
			},
		},
		{
			name: "Only username flag and argUser provided",
			input: dbAndUserTestCase{
				dbnameOpt:    "",
				userOpt:      "flagUser",
				argDB:        "",
				argUser:      "argUser",
				expectedDB:   "",
				expectedUser: "flagUser",
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			actualDB, actualUser := resolveDBAndUser(tc.input.dbnameOpt, tc.input.userOpt, tc.input.argDB, tc.input.argUser)
			assert.Equal(t, tc.input.expectedDB, actualDB, "finalDB does not match expected value")
			assert.Equal(t, tc.input.expectedUser, actualUser, "finalUser does not match expected value")
		})
	}
}

func TestResolvePort(t *testing.T) {
	newCmd := func(t *testing.T) *cobra.Command {
		t.Helper()
		cmd := &cobra.Command{}
		cmd.Flags().Uint16("port", 5432, "port number")
		return cmd
	}

	tests := []struct {
		name     string
		pgxPort  string
		pgPort   string
		explicit string
		expected uint16
	}{
		{name: "uses PGXPORT when flag is omitted", pgxPort: "6543", expected: 6543},
		{name: "uses PGPORT when PGXPORT is absent", pgPort: "6544", expected: 6544},
		{name: "PGXPORT takes precedence over PGPORT", pgxPort: "6543", pgPort: "6544", expected: 6543},
		{name: "explicit port takes precedence over environment", pgPort: "6544", explicit: "5432", expected: 5432},
		{name: "keeps the flag default when no environment port is set", expected: 5432},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("PGXPORT", tt.pgxPort)
			t.Setenv("PGPORT", tt.pgPort)
			cmd := newCmd(t)
			if tt.explicit != "" {
				require.NoError(t, cmd.Flags().Set("port", tt.explicit))
			}

			port, err := cmd.Flags().GetUint16("port")
			require.NoError(t, err)
			assert.Equal(t, tt.expected, resolvePort(cmd, port))
		})
	}
}
