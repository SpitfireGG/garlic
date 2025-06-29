package bubble

import (
	"os"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	api "github.com/spitfiregg/garlic/internal/api/gemini"
	"github.com/spitfiregg/garlic/internal/bubble/chat"
	"github.com/spitfiregg/garlic/window"
)

// TODO: make struct member for more models ( add more models )
// TODO: make user be able to recurse through the chat history

// Viewport dimension
const (
	vpHeight = 40
	vpWidth  = 120
)

type State int

const (
	GreetWindow State = iota
	MainWindow
	LLMwindow
	SettingsWindow
)

// define the main program state
type UI struct {
	textInput textinput.Model
	viewPort  viewport.Model
	height    int
	width     int
}

type App struct {
	ui   *UI
	chat *chat.Session
}

type LLMreponseMsg struct {
	response string
	err      error
}

type DebugModel struct {
	Dump *os.File
}

type Model struct {
	isLLMthinking bool
	api_key       string

	// for window selection
	currentState      State
	LLMSelectorWindow window.LLMmodel
	selectedLLM       string

	//embed the defined structs into the main Model
	App
	UI
	LLMreponseMsg
	DebugModel
}

type TransitionToMain struct{}

func TextInputHandler() textinput.Model {
	ti := textinput.New()
	ti.Placeholder = "Talk to Gemini"
	ti.Focus()
	ti.Cursor.Blink = true
	ti.CharLimit = 512
	ti.Width = 80
	return ti
}

func InitialModel(config *api.AppConfig) Model {

	// jump straight to prompting the model
	vp := viewport.New(vpWidth, vpHeight)
	vp.SetContent("welcome to the Playground...")

	return Model{

		currentState:      GreetWindow,
		LLMSelectorWindow: window.NewModel(),
		isLLMthinking:     false, // initially set the model thinking to be false
		api_key:           config.GeminiDefault.ApiKey,

		// the whole ui of the program
		UI: UI{
			textInput: TextInputHandler(),
			viewPort:  vp,
		},

		// the app itself
		App: App{
			ui:   &UI{},
			chat: chat.NewSession(), // default to new session
		},

		// response from the LLM
		LLMreponseMsg: LLMreponseMsg{
			response: "This is an initial test reponse",
			err:      nil,
		},

		// debugging
		DebugModel: DebugModel{},
	}
}

func Transition(d time.Duration) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(d)
		return TransitionToMain{}
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink)
}
