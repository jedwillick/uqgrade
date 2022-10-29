package main

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	log "github.com/sirupsen/logrus"
)

const (
	MIN_TAB_WIDTH   int = 10
	MIN_WIN_WIDTH   int = 60
	NUM_TABS_SWITCH int = 5
)

var (
	focusedStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	blurredStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	titleStyle          = lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
	cursorStyle         = focusedStyle.Copy()
	noStyle             = lipgloss.NewStyle()
	helpStyle           = blurredStyle.Copy()
	cursorModeHelpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))

	focusedButton = focusedStyle.Copy().Render("[ Submit ]")
	blurredButton = fmt.Sprintf("[ %s ]", blurredStyle.Render("Submit"))

	defaultWidth = 20

	inactiveTabBorder = tabBorderWithBottom("┴", "─", "┴")
	activeTabBorder   = tabBorderWithBottom("┘", " ", "└")
	docStyle          = lipgloss.NewStyle().Padding(1, 2, 1, 2)
	highlightColor    = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
	inactiveTabStyle  = lipgloss.NewStyle().Border(inactiveTabBorder, true).BorderForeground(highlightColor).Padding(0, 1)
	activeTabStyle    = inactiveTabStyle.Copy().Border(activeTabBorder, true)
	windowStyle       = lipgloss.NewStyle().BorderForeground(highlightColor).Padding(1, 2).Align(lipgloss.Left).Border(lipgloss.NormalBorder()).UnsetBorderTop()

	CUTOFFS = map[int]float64{1: 0, 2: 20, 3: 45, 4: 50, 5: 65, 6: 75, 7: 85}
)

type courseModel struct {
	focusIndex int
	inputs     []textinput.Model
	isOverall  bool
	total      float64
	grade      int
	course     Course
}

type tabModel struct {
	Tabs       []string
	TabContent []courseModel
	activeTab  int
	input      textinput.Model
}

func validator(text string) error {
	if strings.HasSuffix(text, "%") {
		text = strings.TrimSuffix(text, "%")
	}
	_, err := strconv.ParseFloat(text, 64)
	return err
}

func initialModel(course Course) courseModel {
	m := courseModel{
		inputs: make([]textinput.Model, len(course.Assessment)),
		course: course,
	}

	var t textinput.Model
	for i, a := range course.Assessment {
		t = textinput.New()
		t.CursorStyle = cursorStyle
		t.Validate = validator

		t.Prompt = fmt.Sprintf("%-20s(%.1f): ", a.Name, a.Weight)
		if i == 0 {
			t.Focus()
			t.PromptStyle = focusedStyle
			t.TextStyle = focusedStyle
		}

		m.inputs[i] = t
	}

	return m
}

func notEditable(text string) error {
	return errors.New("not editable")
}

func initialOverall(courses []Course) courseModel {
	m := courseModel{
		inputs:    make([]textinput.Model, len(courses)),
		isOverall: true,
	}

	var t textinput.Model
	for i, a := range courses {
		t = textinput.New()
		t.CursorStyle = cursorStyle
		// t.Validate = notEditable

		t.Prompt = fmt.Sprintf("%-19s(7.00): ", a.Name)
		if i == 0 {
			t.Focus()
			t.PromptStyle = focusedStyle
			t.TextStyle = focusedStyle
		}

		m.inputs[i] = t
	}

	return m
}

func (m courseModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m *courseModel) updateInputs(msg tea.Msg) tea.Cmd {
	var cmds = make([]tea.Cmd, len(m.inputs))

	// Only text inputs with Focus() set will respond, so it's safe to simply
	// update all of them here without any further logic.
	for i := range m.inputs {
		m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
	}

	return tea.Batch(cmds...)
}

func totalToGrade(total float64) int {
	total = math.Round(total)
	if total < 20 {
		return 1
	} else if total < 45 {
		return 2
	} else if total < 50 {
		return 3
	} else if total < 65 {
		return 4
	} else if total < 75 {
		return 5
	} else if total < 85 {
		return 6
	} else {
		return 7
	}
}

func (m *courseModel) addGrades(assessment []Assessment) (float64, int) {
	total := 0.0
	for i := range m.inputs {
		input := m.inputs[i].Value()
		if input == "" {
			continue
		}
		if strings.HasSuffix(input, "%") {
			input = strings.TrimSuffix(input, "%")
			mark, _ := strconv.ParseFloat(input, 32)
			total += (mark / 100) * assessment[i].Weight
		} else {
			mark, _ := strconv.ParseFloat(input, 32)
			total += mark
		}
	}

	return total, totalToGrade(total)
}

func (m *courseModel) View(t tabModel) string {
	if m.isOverall {
		return m.overallView(t)
	}
	return m.courseView()
}

func (m *courseModel) courseView() string {
	var b strings.Builder
	// b.WriteString(titleStyle.Render("== COMP3506 =="))
	// b.WriteRune('\n')

	for i := range m.inputs {
		b.WriteString(m.inputs[i].View())
		b.WriteRune('\n')
	}

	m.total, m.grade = m.addGrades(m.course.Assessment)

	totalStr := fmt.Sprintf("%-19s%s %.1f", "Total", "(100.0):", m.total)
	gradeStr := fmt.Sprintf("%-23s%s %d", "Current Grade", "(7):", m.grade)
	if m.focusIndex == len(m.inputs) {
		totalStr = focusedStyle.Render(totalStr)
	} else if m.focusIndex == len(m.inputs)+1 {
		gradeStr = focusedStyle.Render(gradeStr)
	}
	b.WriteString(strings.Repeat("-", 35) + "\n")
	fmt.Fprintf(&b, "%s\n%s\n\n", totalStr, gradeStr)
	for i := m.grade + 1; i <= 7; i++ {
		b.WriteString(fmt.Sprintf("To get a %d you need %.2f more percent\n", i, CUTOFFS[i]-m.total))
	}

	return b.String()
}

func (m *courseModel) overallView(t tabModel) string {
	var b strings.Builder

	totalGrade := 0.0
	for i := range m.inputs {
		grade := float64(t.TabContent[i].grade)
		totalGrade += grade
		m.inputs[i].SetValue(fmt.Sprintf("%.2f", grade))
		m.inputs[i].SetCursorMode(textinput.CursorHide)
		b.WriteString(m.inputs[i].View())

		b.WriteRune('\n')
	}
	b.WriteString(strings.Repeat("-", 35) + "\n")
	gradeStr := fmt.Sprintf("%-19s%s %.2f", "Overall Grade", "(7.00):", totalGrade/float64(len(m.inputs)))
	if m.focusIndex == len(m.inputs) {
		gradeStr = focusedStyle.Render(gradeStr)
	}
	fmt.Fprintf(&b, "%s\n\n", gradeStr)

	return b.String()
}

func (m tabModel) Init() tea.Cmd {
	return m.TabContent[m.activeTab].Init()
}

func (m tabModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	tab := &m.TabContent[m.activeTab]
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.input.Focused() {
			switch msg.String() {
			case "ctrl+c", "q":
				return m, tea.Quit
			case "enter", "esc":
				m.input.SetCursorMode(textinput.CursorHide)
				m.input.Blur()
				raw := m.input.Value()
				if raw == "" {
					return m, nil
				}
				codes := strings.FieldsFunc(raw, func(r rune) bool {
					return r == ',' || r == ' '
				})
				for i := range codes {
					codes[i] = strings.TrimSpace(codes[i])
				}

				list, invalid := scrap(codes)
				if len(invalid) > 0 {
					m.input.SetValue(fmt.Sprintf("Invalid course codes: %s", strings.Join(invalid, ", ")))
				} else {
					m.input.SetValue("")
				}
				for _, course := range list {
					index := len(m.Tabs) - 1
					t := textinput.New()
					t.CursorStyle = cursorStyle
					t.CharLimit = 5
					t.Prompt = fmt.Sprintf("%-19s(7.00): ", course.Name)

					m.TabContent[index].inputs = append(m.TabContent[index].inputs, t)

					m.Tabs = append(m.Tabs[:index+1], m.Tabs[index:]...)
					m.Tabs[index] = course.Name
					m.TabContent = append(m.TabContent[:index+1], m.TabContent[index:]...)
					m.TabContent[index] = initialModel(list[0])

					m.activeTab = index
				}

				return m, nil

			}
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			return m, cmd
		}
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "tab", "]":
			m.activeTab++
			if m.activeTab >= len(m.Tabs) {
				m.activeTab = 0
			}
			return m, textinput.Blink
		case "shift+tab", "[":
			m.activeTab--
			if m.activeTab < 0 {
				m.activeTab = len(m.Tabs) - 1
			}
			return m, textinput.Blink
		case "d":
			if m.Tabs[m.activeTab] != "OVERALL" {
				m.Tabs = append(m.Tabs[:m.activeTab], m.Tabs[m.activeTab+1:]...)
				m.TabContent = append(m.TabContent[:m.activeTab], m.TabContent[m.activeTab+1:]...)
				overallTab := &m.TabContent[len(m.TabContent)-1]
				overallTab.inputs = append(overallTab.inputs[:m.activeTab], overallTab.inputs[m.activeTab+1:]...)
				m.activeTab = 0
			}
			return m, textinput.Blink
		case "a":
			m.input.SetValue("")
			m.input.Focus()
			m.input.SetCursorMode(textinput.CursorBlink)
			return m, nil

		// Set focus to next input
		case "enter", "up", "down":
			s := msg.String()
			// Cycle indexes
			if s == "up" {
				tab.focusIndex--
			} else {
				tab.focusIndex++
			}
			offset := 1
			if tab.isOverall {
				offset = 0
			}
			if tab.focusIndex > len(tab.inputs)+offset {
				tab.focusIndex = 0
			} else if tab.focusIndex < 0 {
				tab.focusIndex = len(tab.inputs)
			}
			// cmds := make([]tea.Cmd, len(tab.inputs))
			for i := 0; i <= len(tab.inputs)-1; i++ {
				if i == tab.focusIndex {
					// Set focused state
					tab.inputs[i].Focus()
					tab.inputs[i].PromptStyle = focusedStyle
					tab.inputs[i].TextStyle = focusedStyle
					continue
				}
				// Remove focused state
				tab.inputs[i].Blur()
				tab.inputs[i].PromptStyle = noStyle
				tab.inputs[i].TextStyle = noStyle
			}
			return m, textinput.Blink
		}
	}
	// Handle character input and blinking

	return m, tab.updateInputs(msg)
}

func tabBorderWithBottom(left, middle, right string) lipgloss.Border {
	border := lipgloss.RoundedBorder()
	border.BottomLeft = left
	border.Bottom = middle
	border.BottomRight = right
	return border
}

func (m tabModel) View() string {
	doc := strings.Builder{}

	var renderedTabs []string
	content := m.TabContent[m.activeTab].View(m)
	numTabs := len(m.Tabs)

	for i, t := range m.Tabs {
		var style lipgloss.Style
		isFirst, isLast, isActive := i == 0, i == len(m.Tabs)-1, i == m.activeTab
		if isActive {
			style = activeTabStyle.Copy()
		} else {
			style = inactiveTabStyle.Copy()
		}
		border, _, _, _, _ := style.GetBorder()
		if isFirst && isLast {
			border.BottomLeft = "│"
			border.BottomRight = "│"
			border.Bottom = "─"
		} else if isFirst && isActive {
			border.BottomLeft = "│"
		} else if isFirst && !isActive {
			border.BottomLeft = "├"
		} else if isLast && isActive {
			border.BottomRight = "│"
		} else if isLast && !isActive {
			border.BottomRight = "┤"
		}
		style = style.Border(border)
		if numTabs <= NUM_TABS_SWITCH {
			style = style.Width(MIN_WIN_WIDTH / numTabs)
		} else {
			style = style.Width(MIN_TAB_WIDTH)
		}
		renderedTabs = append(renderedTabs, style.Render(t))
	}

	row := lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...)
	doc.WriteString(row)
	doc.WriteString("\n")
	if numTabs <= NUM_TABS_SWITCH {
		doc.WriteString(windowStyle.Width(MIN_WIN_WIDTH + (2 * (numTabs - 1))).Render(content))
	} else {
		doc.WriteString(windowStyle.Width((MIN_TAB_WIDTH+2)*numTabs - 2).Render(content))
	}
	doc.WriteString("\n")
	doc.WriteString(m.input.View())
	return docStyle.Render(doc.String())
}

func tui(courses []Course) {
	var tabs []string
	var tabContent []courseModel
	for _, c := range courses {
		tabs = append(tabs, c.Name)
		tabContent = append(tabContent, initialModel(c))
	}
	tabs = append(tabs, "OVERALL")
	tabContent = append(tabContent, initialOverall(courses))

	t := textinput.New()
	t.Placeholder = "Course Code(s)"

	m := tabModel{Tabs: tabs, TabContent: tabContent, input: t}
	if err := tea.NewProgram(m).Start(); err != nil {
		log.Fatalln("Error running program:", err)
	}
}
