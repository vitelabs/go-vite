package console

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"

	"github.com/mattn/go-colorable"
	"github.com/peterh/liner"
	"github.com/robertkrimen/otto"
	"github.com/vitelabs/go-vite/cmd/internal/jsre"
	"github.com/vitelabs/go-vite/rpc"
)

var (
	passwordRegexp = regexp.MustCompile(`personal.[nus]`)
	onlyWhitespace = regexp.MustCompile(`^\s*$`)
	exit           = regexp.MustCompile(`^\s*exit\s*;*\s*$`)

	load_typedarray_define_js = "var TA=typedarray;" +
		"var ArrayBuffer= TA.ArrayBuffer;" +
		"var DataView=TA.DataView;" +
		"var Float32Array=TA.Float32Array;" +
		"var Float64Array=TA.Float64Array;" +
		"var Int8Array=TA.Int8Array;" +
		"var Int16Array=TA.Int16Array;" +
		"var Int32Array=TA.Int32Array;" +
		"var Uint8Array=TA.Uint8Array;" +
		"var Uint8ClampedArray=TA.Uint8ClampedArray;" +
		"var Uint16Array=TA.Uint16Array;" +
		"var Uint32Array=TA.Uint32Array; "
)

// HistoryFile is the file within the data directory to store input scrollback.
const HistoryFile = "history"

// DefaultPrompt is the default prompt line prefix to use for user input querying.
const DefaultPrompt = "-> "

// Config is the collection of configurations to fine tune the behavior of the JavaScript console.
type Config struct {
	DataDir  string       // Data directory to store the console history at
	Client   *rpc.Client  // RPC client to execute Vite requests through
	DocRoot  string       // Filesystem path from where to load JavaScript files from
	Prompt   string       // Input prompt prefix string (defaults to DefaultPrompt)
	Preload  []string     // Absolute paths to JavaScript files to preload
	Printer  io.Writer    // Output writer to serialize any display strings to (defaults to os.Stdout)
	Prompter UserPrompter // Input prompter to allow interactive user feedback (defaults to TerminalPrompter)
}

// Console is a JavaScript interpreted runtime environment. It is a fully fledged JavaScript console attached to a running node via an external or in-process RPC client.
type Console struct {
	client   *rpc.Client  // RPC client to execute Vite requests through
	jsre     *jsre.JSRE   // JavaScript runtime environment running the interpreter
	prompt   string       // Input prompt prefix string
	prompter UserPrompter // Input prompter to allow interactive user feedback
	printer  io.Writer    // Output writer to serialize any display strings to
	histPath string       // Absolute path to the console scrollback history
	history  []string     // Scroll history maintained by the console
}

// New initializes a JavaScript interpreted runtime environment and sets defaults with the config struct.
func New(config Config) (*Console, error) {

	// Handle unset config values gracefully
	if config.Prompter == nil {
		config.Prompter = Stdin
	}
	if config.Prompt == "" {
		config.Prompt = DefaultPrompt
	}
	if config.Printer == nil {
		config.Printer = colorable.NewColorableStdout()
	}

	// Initialize the console and return
	console := &Console{
		client:   config.Client,
		jsre:     jsre.New(config.DocRoot, config.Printer),
		prompt:   config.Prompt,
		prompter: config.Prompter,
		printer:  config.Printer,
		histPath: filepath.Join(config.DataDir, HistoryFile),
	}

	if err := os.MkdirAll(config.DataDir, 0700); err != nil {
		return nil, err
	}

	if err := console.init(config.Preload); err != nil {
		return nil, err
	}

	return console, nil
}

// init retrieves the available APIs from the remote RPC provider and initializes the console's JavaScript namespaces based on the exposed modules.
func (c *Console) init(preload []string) error {

	// Initialize the JavaScript <-> Go RPC bridge
	bridge := newBridge(c.client, c.prompter, c.printer)

	c.jsre.Set("b_vite", struct{}{})
	c.jsre.Set("j_assist", struct{}{})

	jViteObj, _ := c.jsre.Get("b_vite")
	jViteObj.Object().Set("send", bridge.Send)
	jViteObj.Object().Set("sendAsync", bridge.Send)

	consoleObj, _ := c.jsre.Get("console")
	consoleObj.Object().Set("log", c.consoleOutput)
	consoleObj.Object().Set("error", c.consoleOutput)

	if err := c.jsre.Compile("polyfill.js", jsre.Polyfill); err != nil {
		return fmt.Errorf("polyfill.js: %v", err)
	}
	if _, err := c.jsre.Run("require('polyfill');"); err != nil {
		return fmt.Errorf("web3 require: %v", err)
	}
	// Load all the internal utility JavaScript libraries
	if err := c.jsre.Compile("vite.js", jsre.Vite_JS); err != nil {
		return fmt.Errorf("vite.js: %v", err)
	}

	if _, err := c.jsre.Run("var vite = require('ViteJS');"); err != nil {
		return fmt.Errorf("ViteJS require: %v", err)
	}

	//The admin.sleep and admin.sleepBlocks are offered by the console and not by the RPC layer.
	c.jsre.Set("admin", struct{}{})
	admin, err := c.jsre.Get("admin")
	if err != nil {
		return err
	}
	if obj := admin.Object(); obj != nil { // make sure the admin api is enabled over the interface
		obj.Set("clearHistory", c.clearHistory)
	}

	//Configure the console's input prompter for scrollback and tab completion
	if c.prompter != nil {
		if content, err := ioutil.ReadFile(c.histPath); err != nil {
			c.prompter.SetHistory(nil)
		} else {
			c.history = strings.Split(string(content), "\n")
			c.prompter.SetHistory(c.history)
		}
		c.prompter.SetWordCompleter(c.AutoCompleteInput)
	}
	return nil
}

// Welcome show summary of current Gvite instance and some metadata about the
// console's available modules.
func (c *Console) Welcome() {

	// Print some generic Gvite metadata
	fmt.Fprintf(c.printer, "Welcome to the Gvite JavaScript console!\n")

	//List all the supported modules for the user to call
	//if apis, err := c.client.SupportedModules(); err == nil {
	//	modules := make([]string, 0, len(apis))
	//	for api, version := range apis {
	//		modules = append(modules, fmt.Sprintf("%s:%s", api, version))
	//	}
	//	sort.Strings(modules)
	//	fmt.Fprintln(c.printer, " modules:", strings.Join(modules, " "))
	//}
	//fmt.Fprintln(c.printer)
}

// Evaluate executes code and pretty prints the result to the specified output stream.
func (c *Console) Evaluate(statement string) error {
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(c.printer, "[native] error: %v\n", r)
		}
	}()
	return c.jsre.Evaluate(statement, c.printer)
}

// Interactive starts an interactive user session, where input is propted from the configured user prompter.
func (c *Console) Interactive() {
	var (
		prompt    = c.prompt          // Current prompt line (used for multi-line inputs)
		indents   = 0                 // Current number of input indents (used for multi-line inputs)
		input     = ""                // Current user input
		scheduler = make(chan string) // Channel to send the next prompt on and receive the input
	)
	// Start a goroutine to listen for promt requests and send back inputs
	go func() {
		for {
			// Read the next user input
			line, err := c.prompter.PromptInput(<-scheduler)
			if err != nil {
				// In case of an error, either clear the prompt or fail
				if err == liner.ErrPromptAborted { // ctrl-C
					prompt, indents, input = c.prompt, 0, ""
					scheduler <- ""
					continue
				}
				close(scheduler)
				return
			}
			// User input retrieved, send for interpretation and loop
			scheduler <- line
		}
	}()
	// Monitor Ctrl-C too in case the input is empty and we need to bail
	abort := make(chan os.Signal, 1)
	signal.Notify(abort, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)

	// Start sending prompts to the user and reading back inputs
	for {
		// Send the next prompt, triggering an input read and process the result
		scheduler <- prompt
		select {
		case <-abort:
			// User forcefully quite the console
			fmt.Fprintln(c.printer, "caught interrupt, exiting")
			return

		case line, ok := <-scheduler:
			// User input was returned by the prompter, handle special cases
			if !ok || (indents <= 0 && exit.MatchString(line)) {
				return
			}
			if onlyWhitespace.MatchString(line) {
				continue
			}
			// Append the line to the input and check for multi-line interpretation
			input += line + "\n"

			indents = countIndents(input)
			if indents <= 0 {
				prompt = c.prompt
			} else {
				prompt = strings.Repeat(".", indents*3) + " "
			}
			// If all the needed lines are present, save the command and run
			if indents <= 0 {
				if len(input) > 0 && input[0] != ' ' && !passwordRegexp.MatchString(input) {
					if command := strings.TrimSpace(input); len(c.history) == 0 || command != c.history[len(c.history)-1] {
						c.history = append(c.history, command)
						if c.prompter != nil {
							c.prompter.AppendHistory(command)
						}
					}
				}
				c.Evaluate(input)
				input = ""
			}
		}
	}
}

// countIndents returns the number of identations for the given input.
// In case of invalid input such as var a = } the result can be negative.
func countIndents(input string) int {
	var (
		indents     = 0
		inString    = false
		strOpenChar = ' '   // keep track of the string open char to allow var str = "I'm ....";
		charEscaped = false // keep track if the previous char was the '\' char, allow var str = "abc\"def";
	)

	for _, c := range input {
		switch c {
		case '\\':
			// indicate next char as escaped when in string and previous char isn't escaping this backslash
			if !charEscaped && inString {
				charEscaped = true
			}
		case '\'', '"':
			if inString && !charEscaped && strOpenChar == c { // end string
				inString = false
			} else if !inString && !charEscaped { // begin string
				inString = true
				strOpenChar = c
			}
			charEscaped = false
		case '{', '(':
			if !inString { // ignore brackets when in string, allow var str = "a{"; without indenting
				indents++
			}
			charEscaped = false
		case '}', ')':
			if !inString {
				indents--
			}
			charEscaped = false
		default:
			charEscaped = false
		}
	}

	return indents
}

// Execute runs the JavaScript file specified as the argument.
func (c *Console) Execute(path string) error {
	return c.jsre.Exec(path)
}

// Stop cleans up the console and terminates the runtime environment.
func (c *Console) Stop(graceful bool) error {
	if err := ioutil.WriteFile(c.histPath, []byte(strings.Join(c.history, "\n")), 0600); err != nil {
		return err
	}
	if err := os.Chmod(c.histPath, 0600); err != nil { // Force 0600, even if it was different previously
		return err
	}
	c.jsre.Stop(graceful)
	return nil
}

// consoleOutput is an override for the console.log and console.error methods to stream the output into the configured output stream instead of stdout.
func (c *Console) consoleOutput(call otto.FunctionCall) otto.Value {
	output := []string{}
	for _, argument := range call.ArgumentList {
		output = append(output, fmt.Sprintf("%v", argument))
	}
	fmt.Fprintln(c.printer, strings.Join(output, " "))
	return otto.Value{}
}

func (c *Console) clearHistory() {
	c.history = nil
	c.prompter.ClearHistory()
	if err := os.Remove(c.histPath); err != nil {
		fmt.Fprintln(c.printer, "can't delete history file:", err)
	} else {
		fmt.Fprintln(c.printer, "history file deleted")
	}
}

// AutoCompleteInput is a pre-assembled word completer to be used by the user input prompter to provide hints to the user about the methods available.
func (c *Console) AutoCompleteInput(line string, pos int) (string, []string, string) {
	// No completions can be provided for empty inputs
	if len(line) == 0 || pos == 0 {
		return "", nil, ""
	}
	// Chunck data to relevant part for autocompletion
	// E.g. in case of nested lines eth.getBalance(eth.coinb<tab><tab>
	start := pos - 1
	for ; start > 0; start-- {
		// Skip all methods and namespaces (i.e. including the dot)
		if line[start] == '.' || (line[start] >= 'a' && line[start] <= 'z') || (line[start] >= 'A' && line[start] <= 'Z') {
			continue
		}
		// Handle web3 in a special way (i.e. other numbers aren't auto completed)
		if start >= 3 && line[start-3:start] == "web3" {
			start -= 3
			continue
		}
		// We've hit an unexpected character, autocomplete form here
		start++
		break
	}
	return line[:start], c.jsre.CompleteKeywords(line[start:pos]), line[pos:]
}
