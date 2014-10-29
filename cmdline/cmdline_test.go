package cmdline

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"
)

var (
	errEcho           = errors.New("echo error")
	flagExtra         bool
	optNoNewline      bool
	flagTopLevelExtra bool
	globalFlag1       string
	globalFlag2       *int64
)

// runEcho is used to implement commands for our tests.
func runEcho(cmd *Command, args []string) error {
	if len(args) == 1 {
		if args[0] == "error" {
			return errEcho
		} else if args[0] == "bad_arg" {
			return cmd.UsageErrorf("Invalid argument %v", args[0])
		}
	}
	if flagExtra {
		args = append(args, "extra")
	}
	if flagTopLevelExtra {
		args = append(args, "tlextra")
	}
	if optNoNewline {
		fmt.Fprint(cmd.Stdout(), args)
	} else {
		fmt.Fprintln(cmd.Stdout(), args)
	}
	return nil
}

// runHello is another function for test commands.
func runHello(cmd *Command, args []string) error {
	if flagTopLevelExtra {
		args = append(args, "tlextra")
	}
	fmt.Fprintln(cmd.Stdout(), strings.Join(append([]string{"Hello"}, args...), " "))
	return nil
}

type testCase struct {
	Args        []string
	Err         error
	Stdout      string
	Stderr      string
	GlobalFlag1 string
	GlobalFlag2 int64
}

func init() {
	os.Setenv("CMDLINE_WIDTH", "80") // make sure the formatting stays the same.
	flag.StringVar(&globalFlag1, "global1", "", "global test flag 1")
	globalFlag2 = flag.Int64("global2", 0, "global test flag 2")
}

func stripOutput(got string) string {
	// The global flags include the flags from the testing package, so strip them
	// out before the comparison.
	re := regexp.MustCompile(" -test[^\n]+\n(?:   [^\n]+\n)+")
	return re.ReplaceAllLiteralString(got, "")
}

func runTestCases(t *testing.T, cmd *Command, tests []testCase) {
	for _, test := range tests {
		// Reset global variables before running each test case.
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		flagExtra = false
		flagTopLevelExtra = false
		optNoNewline = false
		globalFlag1 = ""
		*globalFlag2 = 0

		// Run the execute function and check against expected results.
		cmd.Init(nil, &stdout, &stderr)
		if err := cmd.Execute(test.Args); err != test.Err {
			t.Errorf("Ran with args %q\n GOT error:\n%q\nWANT error:\n%q", test.Args, err, test.Err)
		}
		if got, want := stripOutput(stdout.String()), test.Stdout; got != want {
			t.Errorf("Ran with args %q\n GOT stdout:\n%q\nWANT stdout:\n%q", test.Args, got, want)
		}
		if got, want := stripOutput(stderr.String()), test.Stderr; got != want {
			t.Errorf("Ran with args %q\n GOT stderr:\n%q\nWANT stderr:\n%q", test.Args, got, want)
		}
		if got, want := globalFlag1, test.GlobalFlag1; got != want {
			t.Errorf("global1 flag got %q, want %q", got, want)
		}
		if got, want := *globalFlag2, test.GlobalFlag2; got != want {
			t.Errorf("global2 flag got %q, want %q", got, want)
		}
	}
}

func TestNoCommands(t *testing.T) {
	cmd := &Command{
		Name:  "nocmds",
		Short: "Nocmds is invalid.",
		Long:  "Nocmds has no commands and no run function.",
	}

	var tests = []testCase{
		{
			Args: []string{},
			Err:  ErrUsage,
			Stderr: `ERROR: nocmds: neither Children nor Run is specified

Nocmds has no commands and no run function.

Usage:
   nocmds [ERROR: neither Children nor Run is specified]

The global flags are:
 -global1=
   global test flag 1
 -global2=0
   global test flag 2
`,
		},
		{
			Args: []string{"foo"},
			Err:  ErrUsage,
			Stderr: `ERROR: nocmds: neither Children nor Run is specified

Nocmds has no commands and no run function.

Usage:
   nocmds [ERROR: neither Children nor Run is specified]

The global flags are:
 -global1=
   global test flag 1
 -global2=0
   global test flag 2
`,
		},
	}
	runTestCases(t, cmd, tests)
}

func TestOneCommand(t *testing.T) {
	cmdEcho := &Command{
		Name:  "echo",
		Short: "Print strings on stdout",
		Long: `
Echo prints any strings passed in to stdout.
`,
		Run:      runEcho,
		ArgsName: "[strings]",
		ArgsLong: "[strings] are arbitrary strings that will be echoed.",
	}

	prog := &Command{
		Name:     "onecmd",
		Short:    "Onecmd program.",
		Long:     "Onecmd only has the echo command.",
		Children: []*Command{cmdEcho},
	}

	var tests = []testCase{
		{
			Args: []string{},
			Err:  ErrUsage,
			Stderr: `ERROR: onecmd: no command specified

Onecmd only has the echo command.

Usage:
   onecmd <command>

The onecmd commands are:
   echo        Print strings on stdout
   help        Display help for commands or topics
Run "onecmd help [command]" for command usage.

The global flags are:
 -global1=
   global test flag 1
 -global2=0
   global test flag 2
`,
		},
		{
			Args: []string{"foo"},
			Err:  ErrUsage,
			Stderr: `ERROR: onecmd: unknown command "foo"

Onecmd only has the echo command.

Usage:
   onecmd <command>

The onecmd commands are:
   echo        Print strings on stdout
   help        Display help for commands or topics
Run "onecmd help [command]" for command usage.

The global flags are:
 -global1=
   global test flag 1
 -global2=0
   global test flag 2
`,
		},
		{
			Args: []string{"help"},
			Stdout: `Onecmd only has the echo command.

Usage:
   onecmd <command>

The onecmd commands are:
   echo        Print strings on stdout
   help        Display help for commands or topics
Run "onecmd help [command]" for command usage.

The global flags are:
 -global1=
   global test flag 1
 -global2=0
   global test flag 2
`,
		},
		{
			Args: []string{"help", "echo"},
			Stdout: `Echo prints any strings passed in to stdout.

Usage:
   onecmd echo [strings]

[strings] are arbitrary strings that will be echoed.

The global flags are:
 -global1=
   global test flag 1
 -global2=0
   global test flag 2
`,
		},
		{
			Args: []string{"help", "help"},
			Stdout: `Help with no args displays the usage of the parent command.

Help with args displays the usage of the specified sub-command or help topic.

"help ..." recursively displays help for all commands and topics.

The output is formatted to a target width in runes.  The target width is
determined by checking the environment variable CMDLINE_WIDTH, falling back on
the terminal width from the OS, falling back on 80 chars.  By setting
CMDLINE_WIDTH=x, if x > 0 the width is x, if x < 0 the width is unlimited, and
if x == 0 or is unset one of the fallbacks is used.

Usage:
   onecmd help [flags] [command/topic ...]

[command/topic ...] optionally identifies a specific sub-command or help topic.

The onecmd help flags are:
 -style=text
   The formatting style for help output, either "text" or "godoc".

The global flags are:
 -global1=
   global test flag 1
 -global2=0
   global test flag 2
`,
		},
		{
			Args: []string{"help", "..."},
			Stdout: `Onecmd only has the echo command.

Usage:
   onecmd <command>

The onecmd commands are:
   echo        Print strings on stdout
   help        Display help for commands or topics
Run "onecmd help [command]" for command usage.

The global flags are:
 -global1=
   global test flag 1
 -global2=0
   global test flag 2
================================================================================
Onecmd Echo

Echo prints any strings passed in to stdout.

Usage:
   onecmd echo [strings]

[strings] are arbitrary strings that will be echoed.
================================================================================
Onecmd Help

Help with no args displays the usage of the parent command.

Help with args displays the usage of the specified sub-command or help topic.

"help ..." recursively displays help for all commands and topics.

The output is formatted to a target width in runes.  The target width is
determined by checking the environment variable CMDLINE_WIDTH, falling back on
the terminal width from the OS, falling back on 80 chars.  By setting
CMDLINE_WIDTH=x, if x > 0 the width is x, if x < 0 the width is unlimited, and
if x == 0 or is unset one of the fallbacks is used.

Usage:
   onecmd help [flags] [command/topic ...]

[command/topic ...] optionally identifies a specific sub-command or help topic.

The onecmd help flags are:
 -style=text
   The formatting style for help output, either "text" or "godoc".
`,
		},
		{
			Args: []string{"help", "foo"},
			Err:  ErrUsage,
			Stderr: `ERROR: onecmd: unknown command or topic "foo"

Onecmd only has the echo command.

Usage:
   onecmd <command>

The onecmd commands are:
   echo        Print strings on stdout
   help        Display help for commands or topics
Run "onecmd help [command]" for command usage.

The global flags are:
 -global1=
   global test flag 1
 -global2=0
   global test flag 2
`,
		},
		{
			Args:   []string{"echo", "foo", "bar"},
			Stdout: "[foo bar]\n",
		},
		{
			Args: []string{"echo", "error"},
			Err:  errEcho,
		},
		{
			Args: []string{"echo", "bad_arg"},
			Err:  ErrUsage,
			Stderr: `ERROR: Invalid argument bad_arg

Echo prints any strings passed in to stdout.

Usage:
   onecmd echo [strings]

[strings] are arbitrary strings that will be echoed.

The global flags are:
 -global1=
   global test flag 1
 -global2=0
   global test flag 2
`,
		},
	}
	runTestCases(t, prog, tests)
}

func TestMultiCommands(t *testing.T) {
	cmdEcho := &Command{
		Run:   runEcho,
		Name:  "echo",
		Short: "Print strings on stdout",
		Long: `
Echo prints any strings passed in to stdout.
`,
		ArgsName: "[strings]",
		ArgsLong: "[strings] are arbitrary strings that will be echoed.",
	}
	var cmdEchoOpt = &Command{
		Run:   runEcho,
		Name:  "echoopt",
		Short: "Print strings on stdout, with opts",
		// Try varying number of header/trailer newlines around the long description.
		Long: `Echoopt prints any args passed in to stdout.


`,
		ArgsName: "[args]",
		ArgsLong: "[args] are arbitrary strings that will be echoed.",
	}
	cmdEchoOpt.Flags.BoolVar(&optNoNewline, "n", false, "Do not output trailing newline")

	prog := &Command{
		Name:     "multi",
		Short:    "Multi test command",
		Long:     "Multi has two variants of echo.",
		Children: []*Command{cmdEcho, cmdEchoOpt},
	}
	prog.Flags.BoolVar(&flagExtra, "extra", false, "Print an extra arg")

	var tests = []testCase{
		{
			Args: []string{},
			Err:  ErrUsage,
			Stderr: `ERROR: multi: no command specified

Multi has two variants of echo.

Usage:
   multi [flags] <command>

The multi commands are:
   echo        Print strings on stdout
   echoopt     Print strings on stdout, with opts
   help        Display help for commands or topics
Run "multi help [command]" for command usage.

The multi flags are:
 -extra=false
   Print an extra arg

The global flags are:
 -global1=
   global test flag 1
 -global2=0
   global test flag 2
`,
		},
		{
			Args: []string{"help"},
			Stdout: `Multi has two variants of echo.

Usage:
   multi [flags] <command>

The multi commands are:
   echo        Print strings on stdout
   echoopt     Print strings on stdout, with opts
   help        Display help for commands or topics
Run "multi help [command]" for command usage.

The multi flags are:
 -extra=false
   Print an extra arg

The global flags are:
 -global1=
   global test flag 1
 -global2=0
   global test flag 2
`,
		},
		{
			Args: []string{"help", "..."},
			Stdout: `Multi has two variants of echo.

Usage:
   multi [flags] <command>

The multi commands are:
   echo        Print strings on stdout
   echoopt     Print strings on stdout, with opts
   help        Display help for commands or topics
Run "multi help [command]" for command usage.

The multi flags are:
 -extra=false
   Print an extra arg

The global flags are:
 -global1=
   global test flag 1
 -global2=0
   global test flag 2
================================================================================
Multi Echo

Echo prints any strings passed in to stdout.

Usage:
   multi echo [strings]

[strings] are arbitrary strings that will be echoed.
================================================================================
Multi Echoopt

Echoopt prints any args passed in to stdout.

Usage:
   multi echoopt [flags] [args]

[args] are arbitrary strings that will be echoed.

The multi echoopt flags are:
 -n=false
   Do not output trailing newline
================================================================================
Multi Help

Help with no args displays the usage of the parent command.

Help with args displays the usage of the specified sub-command or help topic.

"help ..." recursively displays help for all commands and topics.

The output is formatted to a target width in runes.  The target width is
determined by checking the environment variable CMDLINE_WIDTH, falling back on
the terminal width from the OS, falling back on 80 chars.  By setting
CMDLINE_WIDTH=x, if x > 0 the width is x, if x < 0 the width is unlimited, and
if x == 0 or is unset one of the fallbacks is used.

Usage:
   multi help [flags] [command/topic ...]

[command/topic ...] optionally identifies a specific sub-command or help topic.

The multi help flags are:
 -style=text
   The formatting style for help output, either "text" or "godoc".
`,
		},
		{
			Args: []string{"help", "echo"},
			Stdout: `Echo prints any strings passed in to stdout.

Usage:
   multi echo [strings]

[strings] are arbitrary strings that will be echoed.

The global flags are:
 -global1=
   global test flag 1
 -global2=0
   global test flag 2
`,
		},
		{
			Args: []string{"help", "echoopt"},
			Stdout: `Echoopt prints any args passed in to stdout.

Usage:
   multi echoopt [flags] [args]

[args] are arbitrary strings that will be echoed.

The multi echoopt flags are:
 -n=false
   Do not output trailing newline

The global flags are:
 -global1=
   global test flag 1
 -global2=0
   global test flag 2
`,
		},
		{
			Args: []string{"help", "foo"},
			Err:  ErrUsage,
			Stderr: `ERROR: multi: unknown command or topic "foo"

Multi has two variants of echo.

Usage:
   multi [flags] <command>

The multi commands are:
   echo        Print strings on stdout
   echoopt     Print strings on stdout, with opts
   help        Display help for commands or topics
Run "multi help [command]" for command usage.

The multi flags are:
 -extra=false
   Print an extra arg

The global flags are:
 -global1=
   global test flag 1
 -global2=0
   global test flag 2
`,
		},
		{
			Args:   []string{"echo", "foo", "bar"},
			Stdout: "[foo bar]\n",
		},
		{
			Args:   []string{"-extra", "echo", "foo", "bar"},
			Stdout: "[foo bar extra]\n",
		},
		{
			Args: []string{"echo", "error"},
			Err:  errEcho,
		},
		{
			Args:   []string{"echoopt", "foo", "bar"},
			Stdout: "[foo bar]\n",
		},
		{
			Args:   []string{"-extra", "echoopt", "foo", "bar"},
			Stdout: "[foo bar extra]\n",
		},
		{
			Args:   []string{"echoopt", "-n", "foo", "bar"},
			Stdout: "[foo bar]",
		},
		{
			Args:   []string{"-extra", "echoopt", "-n", "foo", "bar"},
			Stdout: "[foo bar extra]",
		},
		{
			Args:        []string{"-global1=globalStringValue", "-extra", "echoopt", "-n", "foo", "bar"},
			Stdout:      "[foo bar extra]",
			GlobalFlag1: "globalStringValue",
		},
		{
			Args:        []string{"-global2=42", "echoopt", "-n", "foo", "bar"},
			Stdout:      "[foo bar]",
			GlobalFlag2: 42,
		},
		{
			Args:        []string{"-global1=globalStringOtherValue", "-global2=43", "-extra", "echoopt", "-n", "foo", "bar"},
			Stdout:      "[foo bar extra]",
			GlobalFlag1: "globalStringOtherValue",
			GlobalFlag2: 43,
		},
		{
			Args: []string{"echoopt", "error"},
			Err:  errEcho,
		},
		{
			Args: []string{"echo", "-n", "foo", "bar"},
			Err:  ErrUsage,
			Stderr: `ERROR: multi echo: flag provided but not defined: -n

Echo prints any strings passed in to stdout.

Usage:
   multi echo [strings]

[strings] are arbitrary strings that will be echoed.

The global flags are:
 -global1=
   global test flag 1
 -global2=0
   global test flag 2
`,
		},
		{
			Args: []string{"-nosuchflag", "echo", "foo", "bar"},
			Err:  ErrUsage,
			Stderr: `ERROR: multi: flag provided but not defined: -nosuchflag

Multi has two variants of echo.

Usage:
   multi [flags] <command>

The multi commands are:
   echo        Print strings on stdout
   echoopt     Print strings on stdout, with opts
   help        Display help for commands or topics
Run "multi help [command]" for command usage.

The multi flags are:
 -extra=false
   Print an extra arg

The global flags are:
 -global1=
   global test flag 1
 -global2=0
   global test flag 2
`,
		},
	}
	runTestCases(t, prog, tests)
}

func TestMultiLevelCommands(t *testing.T) {
	cmdEcho := &Command{
		Run:   runEcho,
		Name:  "echo",
		Short: "Print strings on stdout",
		Long: `
Echo prints any strings passed in to stdout.
`,
		ArgsName: "[strings]",
		ArgsLong: "[strings] are arbitrary strings that will be echoed.",
	}
	cmdEchoOpt := &Command{
		Run:   runEcho,
		Name:  "echoopt",
		Short: "Print strings on stdout, with opts",
		// Try varying number of header/trailer newlines around the long description.
		Long: `Echoopt prints any args passed in to stdout.


`,
		ArgsName: "[args]",
		ArgsLong: "[args] are arbitrary strings that will be echoed.",
	}
	cmdEchoOpt.Flags.BoolVar(&optNoNewline, "n", false, "Do not output trailing newline")
	cmdHello := &Command{
		Run:   runHello,
		Name:  "hello",
		Short: "Print strings on stdout preceded by \"Hello\"",
		Long: `
Hello prints any strings passed in to stdout preceded by "Hello".
`,
		ArgsName: "[strings]",
		ArgsLong: "[strings] are arbitrary strings that will be printed.",
	}
	echoProg := &Command{
		Name:     "echoprog",
		Short:    "Set of echo commands",
		Long:     "Echoprog has two variants of echo.",
		Children: []*Command{cmdEcho, cmdEchoOpt},
		Topics: []Topic{
			{Name: "topic3", Short: "Help topic 3 short", Long: "Help topic 3 long."},
		},
	}
	echoProg.Flags.BoolVar(&flagExtra, "extra", false, "Print an extra arg")
	prog := &Command{
		Name:     "toplevelprog",
		Short:    "Top level prog",
		Long:     "Toplevelprog has the echo subprogram and the hello command.",
		Children: []*Command{echoProg, cmdHello},
		Topics: []Topic{
			{Name: "topic1", Short: "Help topic 1 short", Long: "Help topic 1 long."},
			{Name: "topic2", Short: "Help topic 2 short", Long: "Help topic 2 long."},
		},
	}
	prog.Flags.BoolVar(&flagTopLevelExtra, "tlextra", false, "Print an extra arg for all commands")

	var tests = []testCase{
		{
			Args: []string{},
			Err:  ErrUsage,
			Stderr: `ERROR: toplevelprog: no command specified

Toplevelprog has the echo subprogram and the hello command.

Usage:
   toplevelprog [flags] <command>

The toplevelprog commands are:
   echoprog    Set of echo commands
   hello       Print strings on stdout preceded by "Hello"
   help        Display help for commands or topics
Run "toplevelprog help [command]" for command usage.

The toplevelprog additional help topics are:
   topic1      Help topic 1 short
   topic2      Help topic 2 short
Run "toplevelprog help [topic]" for topic details.

The toplevelprog flags are:
 -tlextra=false
   Print an extra arg for all commands

The global flags are:
 -global1=
   global test flag 1
 -global2=0
   global test flag 2
`,
		},
		{
			Args: []string{"help"},
			Stdout: `Toplevelprog has the echo subprogram and the hello command.

Usage:
   toplevelprog [flags] <command>

The toplevelprog commands are:
   echoprog    Set of echo commands
   hello       Print strings on stdout preceded by "Hello"
   help        Display help for commands or topics
Run "toplevelprog help [command]" for command usage.

The toplevelprog additional help topics are:
   topic1      Help topic 1 short
   topic2      Help topic 2 short
Run "toplevelprog help [topic]" for topic details.

The toplevelprog flags are:
 -tlextra=false
   Print an extra arg for all commands

The global flags are:
 -global1=
   global test flag 1
 -global2=0
   global test flag 2
`,
		},
		{
			Args: []string{"help", "..."},
			Stdout: `Toplevelprog has the echo subprogram and the hello command.

Usage:
   toplevelprog [flags] <command>

The toplevelprog commands are:
   echoprog    Set of echo commands
   hello       Print strings on stdout preceded by "Hello"
   help        Display help for commands or topics
Run "toplevelprog help [command]" for command usage.

The toplevelprog additional help topics are:
   topic1      Help topic 1 short
   topic2      Help topic 2 short
Run "toplevelprog help [topic]" for topic details.

The toplevelprog flags are:
 -tlextra=false
   Print an extra arg for all commands

The global flags are:
 -global1=
   global test flag 1
 -global2=0
   global test flag 2
================================================================================
Toplevelprog Echoprog

Echoprog has two variants of echo.

Usage:
   toplevelprog echoprog [flags] <command>

The toplevelprog echoprog commands are:
   echo        Print strings on stdout
   echoopt     Print strings on stdout, with opts

The toplevelprog echoprog additional help topics are:
   topic3      Help topic 3 short

The toplevelprog echoprog flags are:
 -extra=false
   Print an extra arg
================================================================================
Toplevelprog Echoprog Echo

Echo prints any strings passed in to stdout.

Usage:
   toplevelprog echoprog echo [strings]

[strings] are arbitrary strings that will be echoed.
================================================================================
Toplevelprog Echoprog Echoopt

Echoopt prints any args passed in to stdout.

Usage:
   toplevelprog echoprog echoopt [flags] [args]

[args] are arbitrary strings that will be echoed.

The toplevelprog echoprog echoopt flags are:
 -n=false
   Do not output trailing newline
================================================================================
Toplevelprog Echoprog Topic3 - help topic

Help topic 3 long.
================================================================================
Toplevelprog Hello

Hello prints any strings passed in to stdout preceded by "Hello".

Usage:
   toplevelprog hello [strings]

[strings] are arbitrary strings that will be printed.
================================================================================
Toplevelprog Help

Help with no args displays the usage of the parent command.

Help with args displays the usage of the specified sub-command or help topic.

"help ..." recursively displays help for all commands and topics.

The output is formatted to a target width in runes.  The target width is
determined by checking the environment variable CMDLINE_WIDTH, falling back on
the terminal width from the OS, falling back on 80 chars.  By setting
CMDLINE_WIDTH=x, if x > 0 the width is x, if x < 0 the width is unlimited, and
if x == 0 or is unset one of the fallbacks is used.

Usage:
   toplevelprog help [flags] [command/topic ...]

[command/topic ...] optionally identifies a specific sub-command or help topic.

The toplevelprog help flags are:
 -style=text
   The formatting style for help output, either "text" or "godoc".
================================================================================
Toplevelprog Topic1 - help topic

Help topic 1 long.
================================================================================
Toplevelprog Topic2 - help topic

Help topic 2 long.
`,
		},
		{
			Args: []string{"help", "echoprog"},
			Stdout: `Echoprog has two variants of echo.

Usage:
   toplevelprog echoprog [flags] <command>

The toplevelprog echoprog commands are:
   echo        Print strings on stdout
   echoopt     Print strings on stdout, with opts
   help        Display help for commands or topics
Run "toplevelprog echoprog help [command]" for command usage.

The toplevelprog echoprog additional help topics are:
   topic3      Help topic 3 short
Run "toplevelprog echoprog help [topic]" for topic details.

The toplevelprog echoprog flags are:
 -extra=false
   Print an extra arg

The global flags are:
 -global1=
   global test flag 1
 -global2=0
   global test flag 2
`,
		},
		{
			Args: []string{"help", "topic1"},
			Stdout: `Help topic 1 long.
`,
		},
		{
			Args: []string{"help", "topic2"},
			Stdout: `Help topic 2 long.
`,
		},
		{
			Args: []string{"echoprog", "help", "..."},
			Stdout: `Echoprog has two variants of echo.

Usage:
   toplevelprog echoprog [flags] <command>

The toplevelprog echoprog commands are:
   echo        Print strings on stdout
   echoopt     Print strings on stdout, with opts
   help        Display help for commands or topics
Run "toplevelprog echoprog help [command]" for command usage.

The toplevelprog echoprog additional help topics are:
   topic3      Help topic 3 short
Run "toplevelprog echoprog help [topic]" for topic details.

The toplevelprog echoprog flags are:
 -extra=false
   Print an extra arg

The global flags are:
 -global1=
   global test flag 1
 -global2=0
   global test flag 2
================================================================================
Toplevelprog Echoprog Echo

Echo prints any strings passed in to stdout.

Usage:
   toplevelprog echoprog echo [strings]

[strings] are arbitrary strings that will be echoed.
================================================================================
Toplevelprog Echoprog Echoopt

Echoopt prints any args passed in to stdout.

Usage:
   toplevelprog echoprog echoopt [flags] [args]

[args] are arbitrary strings that will be echoed.

The toplevelprog echoprog echoopt flags are:
 -n=false
   Do not output trailing newline
================================================================================
Toplevelprog Echoprog Help

Help with no args displays the usage of the parent command.

Help with args displays the usage of the specified sub-command or help topic.

"help ..." recursively displays help for all commands and topics.

The output is formatted to a target width in runes.  The target width is
determined by checking the environment variable CMDLINE_WIDTH, falling back on
the terminal width from the OS, falling back on 80 chars.  By setting
CMDLINE_WIDTH=x, if x > 0 the width is x, if x < 0 the width is unlimited, and
if x == 0 or is unset one of the fallbacks is used.

Usage:
   toplevelprog echoprog help [flags] [command/topic ...]

[command/topic ...] optionally identifies a specific sub-command or help topic.

The toplevelprog echoprog help flags are:
 -style=text
   The formatting style for help output, either "text" or "godoc".
================================================================================
Toplevelprog Echoprog Topic3 - help topic

Help topic 3 long.
`,
		},
		{
			Args: []string{"echoprog", "help", "echoopt"},
			Stdout: `Echoopt prints any args passed in to stdout.

Usage:
   toplevelprog echoprog echoopt [flags] [args]

[args] are arbitrary strings that will be echoed.

The toplevelprog echoprog echoopt flags are:
 -n=false
   Do not output trailing newline

The global flags are:
 -global1=
   global test flag 1
 -global2=0
   global test flag 2
`,
		},
		{
			Args: []string{"help", "echoprog", "topic3"},
			Stdout: `Help topic 3 long.
`,
		},
		{
			Args: []string{"echoprog", "help", "topic3"},
			Stdout: `Help topic 3 long.
`,
		},
		{
			Args: []string{"help", "hello"},
			Stdout: `Hello prints any strings passed in to stdout preceded by "Hello".

Usage:
   toplevelprog hello [strings]

[strings] are arbitrary strings that will be printed.

The global flags are:
 -global1=
   global test flag 1
 -global2=0
   global test flag 2
`,
		},
		{
			Args: []string{"help", "foo"},
			Err:  ErrUsage,
			Stderr: `ERROR: toplevelprog: unknown command or topic "foo"

Toplevelprog has the echo subprogram and the hello command.

Usage:
   toplevelprog [flags] <command>

The toplevelprog commands are:
   echoprog    Set of echo commands
   hello       Print strings on stdout preceded by "Hello"
   help        Display help for commands or topics
Run "toplevelprog help [command]" for command usage.

The toplevelprog additional help topics are:
   topic1      Help topic 1 short
   topic2      Help topic 2 short
Run "toplevelprog help [topic]" for topic details.

The toplevelprog flags are:
 -tlextra=false
   Print an extra arg for all commands

The global flags are:
 -global1=
   global test flag 1
 -global2=0
   global test flag 2
`,
		},
		{
			Args:   []string{"echoprog", "echo", "foo", "bar"},
			Stdout: "[foo bar]\n",
		},
		{
			Args:   []string{"echoprog", "-extra", "echo", "foo", "bar"},
			Stdout: "[foo bar extra]\n",
		},
		{
			Args: []string{"echoprog", "echo", "error"},
			Err:  errEcho,
		},
		{
			Args:   []string{"echoprog", "echoopt", "foo", "bar"},
			Stdout: "[foo bar]\n",
		},
		{
			Args:   []string{"echoprog", "-extra", "echoopt", "foo", "bar"},
			Stdout: "[foo bar extra]\n",
		},
		{
			Args:   []string{"echoprog", "echoopt", "-n", "foo", "bar"},
			Stdout: "[foo bar]",
		},
		{
			Args:   []string{"echoprog", "-extra", "echoopt", "-n", "foo", "bar"},
			Stdout: "[foo bar extra]",
		},
		{
			Args: []string{"echoprog", "echoopt", "error"},
			Err:  errEcho,
		},
		{
			Args:   []string{"--tlextra", "echoprog", "-extra", "echoopt", "foo", "bar"},
			Stdout: "[foo bar extra tlextra]\n",
		},
		{
			Args:   []string{"hello", "foo", "bar"},
			Stdout: "Hello foo bar\n",
		},
		{
			Args:   []string{"--tlextra", "hello", "foo", "bar"},
			Stdout: "Hello foo bar tlextra\n",
		},
		{
			Args: []string{"hello", "--extra", "foo", "bar"},
			Err:  ErrUsage,
			Stderr: `ERROR: toplevelprog hello: flag provided but not defined: -extra

Hello prints any strings passed in to stdout preceded by "Hello".

Usage:
   toplevelprog hello [strings]

[strings] are arbitrary strings that will be printed.

The global flags are:
 -global1=
   global test flag 1
 -global2=0
   global test flag 2
`,
		},
		{
			Args: []string{"-extra", "echoprog", "echoopt", "foo", "bar"},
			Err:  ErrUsage,
			Stderr: `ERROR: toplevelprog: flag provided but not defined: -extra

Toplevelprog has the echo subprogram and the hello command.

Usage:
   toplevelprog [flags] <command>

The toplevelprog commands are:
   echoprog    Set of echo commands
   hello       Print strings on stdout preceded by "Hello"
   help        Display help for commands or topics
Run "toplevelprog help [command]" for command usage.

The toplevelprog additional help topics are:
   topic1      Help topic 1 short
   topic2      Help topic 2 short
Run "toplevelprog help [topic]" for topic details.

The toplevelprog flags are:
 -tlextra=false
   Print an extra arg for all commands

The global flags are:
 -global1=
   global test flag 1
 -global2=0
   global test flag 2
`,
		},
	}
	runTestCases(t, prog, tests)
}

func TestMultiLevelCommandsOrdering(t *testing.T) {
	cmdHello11 := &Command{
		Name:  "hello11",
		Short: "Print strings on stdout preceded by \"Hello\"",
		Long: `
Hello prints any strings passed in to stdout preceded by "Hello".
`,
		ArgsName: "[strings]",
		ArgsLong: "[strings] are arbitrary strings that will be printed.",
		Run:      runHello,
	}
	cmdHello12 := &Command{
		Name:  "hello12",
		Short: "Print strings on stdout preceded by \"Hello\"",
		Long: `
Hello prints any strings passed in to stdout preceded by "Hello".
`,
		ArgsName: "[strings]",
		ArgsLong: "[strings] are arbitrary strings that will be printed.",
		Run:      runHello,
	}
	cmdHello21 := &Command{
		Name:  "hello21",
		Short: "Print strings on stdout preceded by \"Hello\"",
		Long: `
Hello prints any strings passed in to stdout preceded by "Hello".
`,
		ArgsName: "[strings]",
		ArgsLong: "[strings] are arbitrary strings that will be printed.",
		Run:      runHello,
	}
	cmdHello22 := &Command{
		Name:  "hello22",
		Short: "Print strings on stdout preceded by \"Hello\"",
		Long: `
Hello prints any strings passed in to stdout preceded by "Hello".
`,
		ArgsName: "[strings]",
		ArgsLong: "[strings] are arbitrary strings that will be printed.",
		Run:      runHello,
	}
	cmdHello31 := &Command{
		Name:  "hello31",
		Short: "Print strings on stdout preceded by \"Hello\"",
		Long: `
Hello prints any strings passed in to stdout preceded by "Hello".
`,
		ArgsName: "[strings]",
		ArgsLong: "[strings] are arbitrary strings that will be printed.",
		Run:      runHello,
	}
	cmdHello32 := &Command{
		Name:  "hello32",
		Short: "Print strings on stdout preceded by \"Hello\"",
		Long: `
Hello prints any strings passed in to stdout preceded by "Hello".
`,
		ArgsName: "[strings]",
		ArgsLong: "[strings] are arbitrary strings that will be printed.",
		Run:      runHello,
	}
	progHello3 := &Command{
		Name:     "prog3",
		Short:    "Set of hello commands",
		Long:     "Prog3 has two variants of hello.",
		Children: []*Command{cmdHello31, cmdHello32},
	}
	progHello2 := &Command{
		Name:     "prog2",
		Short:    "Set of hello commands",
		Long:     "Prog2 has two variants of hello and a subprogram prog3.",
		Children: []*Command{cmdHello21, progHello3, cmdHello22},
	}
	progHello1 := &Command{
		Name:     "prog1",
		Short:    "Set of hello commands",
		Long:     "Prog1 has two variants of hello and a subprogram prog2.",
		Children: []*Command{cmdHello11, cmdHello12, progHello2},
	}

	var tests = []testCase{
		{
			Args: []string{},
			Err:  ErrUsage,
			Stderr: `ERROR: prog1: no command specified

Prog1 has two variants of hello and a subprogram prog2.

Usage:
   prog1 <command>

The prog1 commands are:
   hello11     Print strings on stdout preceded by "Hello"
   hello12     Print strings on stdout preceded by "Hello"
   prog2       Set of hello commands
   help        Display help for commands or topics
Run "prog1 help [command]" for command usage.

The global flags are:
 -global1=
   global test flag 1
 -global2=0
   global test flag 2
`,
		},
		{
			Args: []string{"help"},
			Stdout: `Prog1 has two variants of hello and a subprogram prog2.

Usage:
   prog1 <command>

The prog1 commands are:
   hello11     Print strings on stdout preceded by "Hello"
   hello12     Print strings on stdout preceded by "Hello"
   prog2       Set of hello commands
   help        Display help for commands or topics
Run "prog1 help [command]" for command usage.

The global flags are:
 -global1=
   global test flag 1
 -global2=0
   global test flag 2
`,
		},
		{
			Args: []string{"help", "..."},
			Stdout: `Prog1 has two variants of hello and a subprogram prog2.

Usage:
   prog1 <command>

The prog1 commands are:
   hello11     Print strings on stdout preceded by "Hello"
   hello12     Print strings on stdout preceded by "Hello"
   prog2       Set of hello commands
   help        Display help for commands or topics
Run "prog1 help [command]" for command usage.

The global flags are:
 -global1=
   global test flag 1
 -global2=0
   global test flag 2
================================================================================
Prog1 Hello11

Hello prints any strings passed in to stdout preceded by "Hello".

Usage:
   prog1 hello11 [strings]

[strings] are arbitrary strings that will be printed.
================================================================================
Prog1 Hello12

Hello prints any strings passed in to stdout preceded by "Hello".

Usage:
   prog1 hello12 [strings]

[strings] are arbitrary strings that will be printed.
================================================================================
Prog1 Prog2

Prog2 has two variants of hello and a subprogram prog3.

Usage:
   prog1 prog2 <command>

The prog1 prog2 commands are:
   hello21     Print strings on stdout preceded by "Hello"
   prog3       Set of hello commands
   hello22     Print strings on stdout preceded by "Hello"
================================================================================
Prog1 Prog2 Hello21

Hello prints any strings passed in to stdout preceded by "Hello".

Usage:
   prog1 prog2 hello21 [strings]

[strings] are arbitrary strings that will be printed.
================================================================================
Prog1 Prog2 Prog3

Prog3 has two variants of hello.

Usage:
   prog1 prog2 prog3 <command>

The prog1 prog2 prog3 commands are:
   hello31     Print strings on stdout preceded by "Hello"
   hello32     Print strings on stdout preceded by "Hello"
================================================================================
Prog1 Prog2 Prog3 Hello31

Hello prints any strings passed in to stdout preceded by "Hello".

Usage:
   prog1 prog2 prog3 hello31 [strings]

[strings] are arbitrary strings that will be printed.
================================================================================
Prog1 Prog2 Prog3 Hello32

Hello prints any strings passed in to stdout preceded by "Hello".

Usage:
   prog1 prog2 prog3 hello32 [strings]

[strings] are arbitrary strings that will be printed.
================================================================================
Prog1 Prog2 Hello22

Hello prints any strings passed in to stdout preceded by "Hello".

Usage:
   prog1 prog2 hello22 [strings]

[strings] are arbitrary strings that will be printed.
================================================================================
Prog1 Help

Help with no args displays the usage of the parent command.

Help with args displays the usage of the specified sub-command or help topic.

"help ..." recursively displays help for all commands and topics.

The output is formatted to a target width in runes.  The target width is
determined by checking the environment variable CMDLINE_WIDTH, falling back on
the terminal width from the OS, falling back on 80 chars.  By setting
CMDLINE_WIDTH=x, if x > 0 the width is x, if x < 0 the width is unlimited, and
if x == 0 or is unset one of the fallbacks is used.

Usage:
   prog1 help [flags] [command/topic ...]

[command/topic ...] optionally identifies a specific sub-command or help topic.

The prog1 help flags are:
 -style=text
   The formatting style for help output, either "text" or "godoc".
`,
		},
		{
			Args: []string{"prog2", "help", "..."},
			Stdout: `Prog2 has two variants of hello and a subprogram prog3.

Usage:
   prog1 prog2 <command>

The prog1 prog2 commands are:
   hello21     Print strings on stdout preceded by "Hello"
   prog3       Set of hello commands
   hello22     Print strings on stdout preceded by "Hello"
   help        Display help for commands or topics
Run "prog1 prog2 help [command]" for command usage.

The global flags are:
 -global1=
   global test flag 1
 -global2=0
   global test flag 2
================================================================================
Prog1 Prog2 Hello21

Hello prints any strings passed in to stdout preceded by "Hello".

Usage:
   prog1 prog2 hello21 [strings]

[strings] are arbitrary strings that will be printed.
================================================================================
Prog1 Prog2 Prog3

Prog3 has two variants of hello.

Usage:
   prog1 prog2 prog3 <command>

The prog1 prog2 prog3 commands are:
   hello31     Print strings on stdout preceded by "Hello"
   hello32     Print strings on stdout preceded by "Hello"
================================================================================
Prog1 Prog2 Prog3 Hello31

Hello prints any strings passed in to stdout preceded by "Hello".

Usage:
   prog1 prog2 prog3 hello31 [strings]

[strings] are arbitrary strings that will be printed.
================================================================================
Prog1 Prog2 Prog3 Hello32

Hello prints any strings passed in to stdout preceded by "Hello".

Usage:
   prog1 prog2 prog3 hello32 [strings]

[strings] are arbitrary strings that will be printed.
================================================================================
Prog1 Prog2 Hello22

Hello prints any strings passed in to stdout preceded by "Hello".

Usage:
   prog1 prog2 hello22 [strings]

[strings] are arbitrary strings that will be printed.
================================================================================
Prog1 Prog2 Help

Help with no args displays the usage of the parent command.

Help with args displays the usage of the specified sub-command or help topic.

"help ..." recursively displays help for all commands and topics.

The output is formatted to a target width in runes.  The target width is
determined by checking the environment variable CMDLINE_WIDTH, falling back on
the terminal width from the OS, falling back on 80 chars.  By setting
CMDLINE_WIDTH=x, if x > 0 the width is x, if x < 0 the width is unlimited, and
if x == 0 or is unset one of the fallbacks is used.

Usage:
   prog1 prog2 help [flags] [command/topic ...]

[command/topic ...] optionally identifies a specific sub-command or help topic.

The prog1 prog2 help flags are:
 -style=text
   The formatting style for help output, either "text" or "godoc".
`,
		},
		{
			Args: []string{"prog2", "prog3", "help", "..."},
			Stdout: `Prog3 has two variants of hello.

Usage:
   prog1 prog2 prog3 <command>

The prog1 prog2 prog3 commands are:
   hello31     Print strings on stdout preceded by "Hello"
   hello32     Print strings on stdout preceded by "Hello"
   help        Display help for commands or topics
Run "prog1 prog2 prog3 help [command]" for command usage.

The global flags are:
 -global1=
   global test flag 1
 -global2=0
   global test flag 2
================================================================================
Prog1 Prog2 Prog3 Hello31

Hello prints any strings passed in to stdout preceded by "Hello".

Usage:
   prog1 prog2 prog3 hello31 [strings]

[strings] are arbitrary strings that will be printed.
================================================================================
Prog1 Prog2 Prog3 Hello32

Hello prints any strings passed in to stdout preceded by "Hello".

Usage:
   prog1 prog2 prog3 hello32 [strings]

[strings] are arbitrary strings that will be printed.
================================================================================
Prog1 Prog2 Prog3 Help

Help with no args displays the usage of the parent command.

Help with args displays the usage of the specified sub-command or help topic.

"help ..." recursively displays help for all commands and topics.

The output is formatted to a target width in runes.  The target width is
determined by checking the environment variable CMDLINE_WIDTH, falling back on
the terminal width from the OS, falling back on 80 chars.  By setting
CMDLINE_WIDTH=x, if x > 0 the width is x, if x < 0 the width is unlimited, and
if x == 0 or is unset one of the fallbacks is used.

Usage:
   prog1 prog2 prog3 help [flags] [command/topic ...]

[command/topic ...] optionally identifies a specific sub-command or help topic.

The prog1 prog2 prog3 help flags are:
 -style=text
   The formatting style for help output, either "text" or "godoc".
`,
		},
		{
			Args: []string{"help", "prog2", "prog3", "..."},
			Stdout: `Prog3 has two variants of hello.

Usage:
   prog1 prog2 prog3 <command>

The prog1 prog2 prog3 commands are:
   hello31     Print strings on stdout preceded by "Hello"
   hello32     Print strings on stdout preceded by "Hello"
   help        Display help for commands or topics
Run "prog1 prog2 prog3 help [command]" for command usage.

The global flags are:
 -global1=
   global test flag 1
 -global2=0
   global test flag 2
================================================================================
Prog1 Prog2 Prog3 Hello31

Hello prints any strings passed in to stdout preceded by "Hello".

Usage:
   prog1 prog2 prog3 hello31 [strings]

[strings] are arbitrary strings that will be printed.
================================================================================
Prog1 Prog2 Prog3 Hello32

Hello prints any strings passed in to stdout preceded by "Hello".

Usage:
   prog1 prog2 prog3 hello32 [strings]

[strings] are arbitrary strings that will be printed.
================================================================================
Prog1 Prog2 Prog3 Help

Help with no args displays the usage of the parent command.

Help with args displays the usage of the specified sub-command or help topic.

"help ..." recursively displays help for all commands and topics.

The output is formatted to a target width in runes.  The target width is
determined by checking the environment variable CMDLINE_WIDTH, falling back on
the terminal width from the OS, falling back on 80 chars.  By setting
CMDLINE_WIDTH=x, if x > 0 the width is x, if x < 0 the width is unlimited, and
if x == 0 or is unset one of the fallbacks is used.

Usage:
   prog1 prog2 prog3 help [flags] [command/topic ...]

[command/topic ...] optionally identifies a specific sub-command or help topic.

The prog1 prog2 prog3 help flags are:
 -style=text
   The formatting style for help output, either "text" or "godoc".
`,
		},
		{
			Args: []string{"help", "-style=godoc", "..."},
			Stdout: `Prog1 has two variants of hello and a subprogram prog2.

Usage:
   prog1 <command>

The prog1 commands are:
   hello11     Print strings on stdout preceded by "Hello"
   hello12     Print strings on stdout preceded by "Hello"
   prog2       Set of hello commands
   help        Display help for commands or topics
Run "prog1 help [command]" for command usage.

The global flags are:
 -global1=
   global test flag 1
 -global2=0
   global test flag 2

Prog1 Hello11

Hello prints any strings passed in to stdout preceded by "Hello".

Usage:
   prog1 hello11 [strings]

[strings] are arbitrary strings that will be printed.

Prog1 Hello12

Hello prints any strings passed in to stdout preceded by "Hello".

Usage:
   prog1 hello12 [strings]

[strings] are arbitrary strings that will be printed.

Prog1 Prog2

Prog2 has two variants of hello and a subprogram prog3.

Usage:
   prog1 prog2 <command>

The prog1 prog2 commands are:
   hello21     Print strings on stdout preceded by "Hello"
   prog3       Set of hello commands
   hello22     Print strings on stdout preceded by "Hello"

Prog1 Prog2 Hello21

Hello prints any strings passed in to stdout preceded by "Hello".

Usage:
   prog1 prog2 hello21 [strings]

[strings] are arbitrary strings that will be printed.

Prog1 Prog2 Prog3

Prog3 has two variants of hello.

Usage:
   prog1 prog2 prog3 <command>

The prog1 prog2 prog3 commands are:
   hello31     Print strings on stdout preceded by "Hello"
   hello32     Print strings on stdout preceded by "Hello"

Prog1 Prog2 Prog3 Hello31

Hello prints any strings passed in to stdout preceded by "Hello".

Usage:
   prog1 prog2 prog3 hello31 [strings]

[strings] are arbitrary strings that will be printed.

Prog1 Prog2 Prog3 Hello32

Hello prints any strings passed in to stdout preceded by "Hello".

Usage:
   prog1 prog2 prog3 hello32 [strings]

[strings] are arbitrary strings that will be printed.

Prog1 Prog2 Hello22

Hello prints any strings passed in to stdout preceded by "Hello".

Usage:
   prog1 prog2 hello22 [strings]

[strings] are arbitrary strings that will be printed.

Prog1 Help

Help with no args displays the usage of the parent command.

Help with args displays the usage of the specified sub-command or help topic.

"help ..." recursively displays help for all commands and topics.

The output is formatted to a target width in runes.  The target width is
determined by checking the environment variable CMDLINE_WIDTH, falling back on
the terminal width from the OS, falling back on 80 chars.  By setting
CMDLINE_WIDTH=x, if x > 0 the width is x, if x < 0 the width is unlimited, and
if x == 0 or is unset one of the fallbacks is used.

Usage:
   prog1 help [flags] [command/topic ...]

[command/topic ...] optionally identifies a specific sub-command or help topic.

The prog1 help flags are:
 -style=text
   The formatting style for help output, either "text" or "godoc".
`,
		},
	}

	runTestCases(t, progHello1, tests)
}

func TestCommandAndArgs(t *testing.T) {
	cmdEcho := &Command{
		Name:  "echo",
		Short: "Print strings on stdout",
		Long: `
Echo prints any strings passed in to stdout.
`,
		Run:      runEcho,
		ArgsName: "[strings]",
		ArgsLong: "[strings] are arbitrary strings that will be echoed.",
	}

	prog := &Command{
		Name:     "cmdargs",
		Short:    "Cmdargs program.",
		Long:     "Cmdargs has the echo command and a Run function with args.",
		Children: []*Command{cmdEcho},
		Run:      runHello,
		ArgsName: "[strings]",
		ArgsLong: "[strings] are arbitrary strings that will be printed.",
	}

	var tests = []testCase{
		{
			Args:   []string{},
			Stdout: "Hello\n",
		},
		{
			Args:   []string{"foo"},
			Stdout: "Hello foo\n",
		},
		{
			Args: []string{"help"},
			Stdout: `Cmdargs has the echo command and a Run function with args.

Usage:
   cmdargs <command>
   cmdargs [strings]

The cmdargs commands are:
   echo        Print strings on stdout
   help        Display help for commands or topics
Run "cmdargs help [command]" for command usage.

[strings] are arbitrary strings that will be printed.

The global flags are:
 -global1=
   global test flag 1
 -global2=0
   global test flag 2
`,
		},
		{
			Args: []string{"help", "echo"},
			Stdout: `Echo prints any strings passed in to stdout.

Usage:
   cmdargs echo [strings]

[strings] are arbitrary strings that will be echoed.

The global flags are:
 -global1=
   global test flag 1
 -global2=0
   global test flag 2
`,
		},
		{
			Args: []string{"help", "..."},
			Stdout: `Cmdargs has the echo command and a Run function with args.

Usage:
   cmdargs <command>
   cmdargs [strings]

The cmdargs commands are:
   echo        Print strings on stdout
   help        Display help for commands or topics
Run "cmdargs help [command]" for command usage.

[strings] are arbitrary strings that will be printed.

The global flags are:
 -global1=
   global test flag 1
 -global2=0
   global test flag 2
================================================================================
Cmdargs Echo

Echo prints any strings passed in to stdout.

Usage:
   cmdargs echo [strings]

[strings] are arbitrary strings that will be echoed.
================================================================================
Cmdargs Help

Help with no args displays the usage of the parent command.

Help with args displays the usage of the specified sub-command or help topic.

"help ..." recursively displays help for all commands and topics.

The output is formatted to a target width in runes.  The target width is
determined by checking the environment variable CMDLINE_WIDTH, falling back on
the terminal width from the OS, falling back on 80 chars.  By setting
CMDLINE_WIDTH=x, if x > 0 the width is x, if x < 0 the width is unlimited, and
if x == 0 or is unset one of the fallbacks is used.

Usage:
   cmdargs help [flags] [command/topic ...]

[command/topic ...] optionally identifies a specific sub-command or help topic.

The cmdargs help flags are:
 -style=text
   The formatting style for help output, either "text" or "godoc".
`,
		},
		{
			Args: []string{"help", "foo"},
			Err:  ErrUsage,
			Stderr: `ERROR: cmdargs: unknown command or topic "foo"

Cmdargs has the echo command and a Run function with args.

Usage:
   cmdargs <command>
   cmdargs [strings]

The cmdargs commands are:
   echo        Print strings on stdout
   help        Display help for commands or topics
Run "cmdargs help [command]" for command usage.

[strings] are arbitrary strings that will be printed.

The global flags are:
 -global1=
   global test flag 1
 -global2=0
   global test flag 2
`,
		},
		{
			Args:   []string{"echo", "foo", "bar"},
			Stdout: "[foo bar]\n",
		},
		{
			Args: []string{"echo", "error"},
			Err:  errEcho,
		},
		{
			Args: []string{"echo", "bad_arg"},
			Err:  ErrUsage,
			Stderr: `ERROR: Invalid argument bad_arg

Echo prints any strings passed in to stdout.

Usage:
   cmdargs echo [strings]

[strings] are arbitrary strings that will be echoed.

The global flags are:
 -global1=
   global test flag 1
 -global2=0
   global test flag 2
`,
		},
	}
	runTestCases(t, prog, tests)
}

func TestCommandAndRunNoArgs(t *testing.T) {
	cmdEcho := &Command{
		Name:  "echo",
		Short: "Print strings on stdout",
		Long: `
Echo prints any strings passed in to stdout.
`,
		Run:      runEcho,
		ArgsName: "[strings]",
		ArgsLong: "[strings] are arbitrary strings that will be echoed.",
	}

	prog := &Command{
		Name:     "cmdrun",
		Short:    "Cmdrun program.",
		Long:     "Cmdrun has the echo command and a Run function with no args.",
		Children: []*Command{cmdEcho},
		Run:      runHello,
	}

	var tests = []testCase{
		{
			Args:   []string{},
			Stdout: "Hello\n",
		},
		{
			Args: []string{"foo"},
			Err:  ErrUsage,
			Stderr: `ERROR: cmdrun: unknown command "foo"

Cmdrun has the echo command and a Run function with no args.

Usage:
   cmdrun <command>
   cmdrun

The cmdrun commands are:
   echo        Print strings on stdout
   help        Display help for commands or topics
Run "cmdrun help [command]" for command usage.

The global flags are:
 -global1=
   global test flag 1
 -global2=0
   global test flag 2
`,
		},
		{
			Args: []string{"help"},
			Stdout: `Cmdrun has the echo command and a Run function with no args.

Usage:
   cmdrun <command>
   cmdrun

The cmdrun commands are:
   echo        Print strings on stdout
   help        Display help for commands or topics
Run "cmdrun help [command]" for command usage.

The global flags are:
 -global1=
   global test flag 1
 -global2=0
   global test flag 2
`,
		},
		{
			Args: []string{"help", "echo"},
			Stdout: `Echo prints any strings passed in to stdout.

Usage:
   cmdrun echo [strings]

[strings] are arbitrary strings that will be echoed.

The global flags are:
 -global1=
   global test flag 1
 -global2=0
   global test flag 2
`,
		},
		{
			Args: []string{"help", "..."},
			Stdout: `Cmdrun has the echo command and a Run function with no args.

Usage:
   cmdrun <command>
   cmdrun

The cmdrun commands are:
   echo        Print strings on stdout
   help        Display help for commands or topics
Run "cmdrun help [command]" for command usage.

The global flags are:
 -global1=
   global test flag 1
 -global2=0
   global test flag 2
================================================================================
Cmdrun Echo

Echo prints any strings passed in to stdout.

Usage:
   cmdrun echo [strings]

[strings] are arbitrary strings that will be echoed.
================================================================================
Cmdrun Help

Help with no args displays the usage of the parent command.

Help with args displays the usage of the specified sub-command or help topic.

"help ..." recursively displays help for all commands and topics.

The output is formatted to a target width in runes.  The target width is
determined by checking the environment variable CMDLINE_WIDTH, falling back on
the terminal width from the OS, falling back on 80 chars.  By setting
CMDLINE_WIDTH=x, if x > 0 the width is x, if x < 0 the width is unlimited, and
if x == 0 or is unset one of the fallbacks is used.

Usage:
   cmdrun help [flags] [command/topic ...]

[command/topic ...] optionally identifies a specific sub-command or help topic.

The cmdrun help flags are:
 -style=text
   The formatting style for help output, either "text" or "godoc".
`,
		},
		{
			Args: []string{"help", "foo"},
			Err:  ErrUsage,
			Stderr: `ERROR: cmdrun: unknown command or topic "foo"

Cmdrun has the echo command and a Run function with no args.

Usage:
   cmdrun <command>
   cmdrun

The cmdrun commands are:
   echo        Print strings on stdout
   help        Display help for commands or topics
Run "cmdrun help [command]" for command usage.

The global flags are:
 -global1=
   global test flag 1
 -global2=0
   global test flag 2
`,
		},
		{
			Args:   []string{"echo", "foo", "bar"},
			Stdout: "[foo bar]\n",
		},
		{
			Args: []string{"echo", "error"},
			Err:  errEcho,
		},
		{
			Args: []string{"echo", "bad_arg"},
			Err:  ErrUsage,
			Stderr: `ERROR: Invalid argument bad_arg

Echo prints any strings passed in to stdout.

Usage:
   cmdrun echo [strings]

[strings] are arbitrary strings that will be echoed.

The global flags are:
 -global1=
   global test flag 1
 -global2=0
   global test flag 2
`,
		},
	}
	runTestCases(t, prog, tests)
}

func TestLongCommandsHelp(t *testing.T) {
	cmdLong := &Command{
		Name:  "thisisaverylongcommand",
		Short: "the short description of the very long command is very long, and will have to be wrapped",
		Long:  "The long description of the very long command is also very long, and will similarly have to be wrapped",
		Run:   runEcho,
	}
	cmdShort := &Command{
		Name:  "x",
		Short: "description of short command.",
		Long:  "blah blah blah",
		Run:   runEcho,
	}
	prog := &Command{
		Name:     "program",
		Short:    "Test help strings when there are long commands.",
		Long:     "Test help strings when there are long commands.",
		Children: []*Command{cmdShort, cmdLong},
	}
	var tests = []testCase{
		{
			Args: []string{"help"},
			Stdout: `Test help strings when there are long commands.

Usage:
   program <command>

The program commands are:
   x                      description of short command.
   thisisaverylongcommand the short description of the very long command is very
                          long, and will have to be wrapped
   help                   Display help for commands or topics
Run "program help [command]" for command usage.

The global flags are:
 -global1=
   global test flag 1
 -global2=0
   global test flag 2
`,
		},
		{
			Args: []string{"help", "thisisaverylongcommand"},
			Stdout: `The long description of the very long command is also very long, and will
similarly have to be wrapped

Usage:
   program thisisaverylongcommand

The global flags are:
 -global1=
   global test flag 1
 -global2=0
   global test flag 2
`,
		},
	}
	runTestCases(t, prog, tests)
}
