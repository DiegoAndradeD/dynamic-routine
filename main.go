package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ==========================================
// 1. Tipos e Modelos de Domínio
// ==========================================

type Category string

const (
	CategoryBody    Category = "body"
	CategoryFocus   Category = "focus"
	CategoryLeisure Category = "leisure"
)

type Activity struct {
	ID                 string   `json:"id"`
	Name               string   `json:"name"`
	Category           Category `json:"category"`
	TargetPerWeek      int      `json:"targetPerWeek"`
	RecommendedMinutes int      `json:"recommendedMinutes"`
	Weight             int      `json:"weight"`
}

type QueueItem struct {
	InstanceID string `json:"instanceId"`
	ActivityID string `json:"activityId"`
}

type ActivityLog struct {
	ID          string    `json:"id"`
	ActivityID  string    `json:"activityId"`
	CompletedAt time.Time `json:"completedAt"`
	Type        string    `json:"type"` // "full" ou "micro"
}

type OperationalRule struct {
	ID          string `json:"id"`
	Category    string `json:"category,omitempty"`
	Trigger     string `json:"trigger"`
	Instruction string `json:"instruction"`
}

type RoutineState struct {
	Version    int                 `json:"version"`
	Queue      []QueueItem         `json:"queue"`
	Activities map[string]Activity `json:"activities"`
	Logs       []ActivityLog       `json:"logs"`
	Rules      []OperationalRule   `json:"rules"`
}

type WeeklyStat struct {
	Full  int
	Micro int
	Total int
}

// ==========================================
// 2. Estado Inicial & Persistência em Disco
// ==========================================

const stateFileName = "routine_state.json"

func defaultRoutineState() RoutineState {
	return RoutineState{
		Version: 1,
		Activities: map[string]Activity{
			"workout": {
				ID: "workout", Name: "Treino / Exercício", Category: CategoryBody,
				TargetPerWeek: 5, RecommendedMinutes: 90, Weight: 5,
			},
			"rust": {
				ID: "rust", Name: "Estudo de Rust", Category: CategoryFocus,
				TargetPerWeek: 4, RecommendedMinutes: 60, Weight: 4,
			},
			"general_studies": {
				ID: "general_studies", Name: "Estudos Gerais", Category: CategoryFocus,
				TargetPerWeek: 3, RecommendedMinutes: 90, Weight: 3,
			},
			"instrument": {
				ID: "instrument", Name: "Instrumento Musical", Category: CategoryLeisure,
				TargetPerWeek: 3, RecommendedMinutes: 45, Weight: 3,
			},
			"reading": {
				ID: "reading", Name: "Leitura", Category: CategoryLeisure,
				TargetPerWeek: 2, RecommendedMinutes: 45, Weight: 2,
			},
			"videogame": {
				ID: "videogame", Name: "Videogame", Category: CategoryLeisure,
				TargetPerWeek: 2, RecommendedMinutes: 120, Weight: 2,
			},
		},
		Queue: []QueueItem{
			{InstanceID: "q-1", ActivityID: "workout"},
			{InstanceID: "q-2", ActivityID: "rust"},
			{InstanceID: "q-3", ActivityID: "instrument"},
			{InstanceID: "q-4", ActivityID: "workout"},
			{InstanceID: "q-5", ActivityID: "general_studies"},
			{InstanceID: "q-6", ActivityID: "reading"},
			{InstanceID: "q-7", ActivityID: "workout"},
			{InstanceID: "q-8", ActivityID: "rust"},
			{InstanceID: "q-9", ActivityID: "videogame"},
			{InstanceID: "q-10", ActivityID: "workout"},
			{InstanceID: "q-11", ActivityID: "general_studies"},
			{InstanceID: "q-12", ActivityID: "instrument"},
			{InstanceID: "q-13", ActivityID: "workout"},
			{InstanceID: "q-14", ActivityID: "rust"},
			{InstanceID: "q-15", ActivityID: "reading"},
			{InstanceID: "q-16", ActivityID: "general_studies"},
			{InstanceID: "q-17", ActivityID: "videogame"},
		},
		Logs: []ActivityLog{},
		Rules: []OperationalRule{
			{
				ID:          "r-1",
				Trigger:     "Dia começou tarde ou atrasou (ex: ativo após 14h)",
				Instruction: "Trabalho (8h) + Treino express (45-60m) + Microbloco de Rust (20m). Pule o restante para não invadir o sono.",
			},
			{
				ID:          "r-2",
				Trigger:     "Cérebro frito pós-trabalho (Saturação mental total)",
				Instruction: "Não abra código nem teoria. Vá direto para o Treino e depois Instrumento ou descanso passivo.",
			},
			{
				ID:          "r-3",
				Trigger:     "Sem tempo para treinar 2 horas completas",
				Instruction: "Faça versão reduzida de 40 a 50 min. Não zere o treino; estímulo rápido conta como presença.",
			},
			{
				ID:          "r-4",
				Trigger:     "Rust travado ou conceito denso sem foco",
				Instruction: "Use o Microbloco (20m) para revisar sintaxe básica ou código de ontem. Registre como micro e avance a fila.",
			},
			{
				ID:          "r-5",
				Trigger:     "Semana atípica / Não fez quase nada por dias",
				Instruction: "Não faça sessões duplas para compensar. A esteira estava congelada esperando você; continue do topo.",
			},
		},
	}
}

func loadOrCreateState(path string) (RoutineState, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		st := defaultRoutineState()
		data, err := json.MarshalIndent(st, "", "  ")
		if err != nil {
			return st, err
		}
		_ = os.WriteFile(path, data, 0644)
		return st, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return defaultRoutineState(), err
	}

	var st RoutineState
	if err := json.Unmarshal(data, &st); err != nil {
		return defaultRoutineState(), err
	}
	return st, nil
}

func saveState(path string, st RoutineState) error {
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// ==========================================
// 3. Estilos Lip Gloss
// ==========================================

var (
	colorAmber   = lipgloss.Color("#F59E0B")
	colorEmerald = lipgloss.Color("#10B981")
	colorRose    = lipgloss.Color("#F43F5E")
	colorPurple  = lipgloss.Color("#A855F7")
	colorCyan    = lipgloss.Color("#06B6D4")
	colorMuted   = lipgloss.Color("#737373")
	colorDark    = lipgloss.Color("#171717")
	colorBorder  = lipgloss.Color("#262626")

	headerTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FAFAFA"))

	headerSubStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	cardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorAmber).
			Padding(1, 2).
			MarginBottom(1)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(1, 2).
			MarginBottom(1)

	badgeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(colorDark).
			Padding(0, 1).
			Bold(true)

	keyStyle = lipgloss.NewStyle().
			Foreground(colorAmber).
			Bold(true)
)

func getCategoryColor(cat Category) lipgloss.Color {
	switch cat {
	case CategoryBody:
		return colorEmerald
	case CategoryFocus:
		return colorAmber
	case CategoryLeisure:
		return colorPurple
	default:
		return colorCyan
	}
}

// ==========================================
// 4. Modelo Bubble Tea & Lógica
// ==========================================

type viewMode int

const (
	viewDashboard viewMode = iota
	viewRules
)

type model struct {
	state      RoutineState
	filePath   string
	mode       viewMode
	width      int
	height     int
	statusMsg  string
	msgTimer   int
	ruleCursor int
}

func initialModel(filePath string) model {
	st, _ := loadOrCreateState(filePath)
	return model{
		state:    st,
		filePath: filePath,
		mode:     viewDashboard,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

// CompleteCurrent roda a esteira e persiste log
func (m *model) completeCurrent(sessionType string) {
	if len(m.state.Queue) == 0 {
		return
	}

	current := m.state.Queue[0]
	rest := m.state.Queue[1:]

	log := ActivityLog{
		ID:          fmt.Sprintf("log-%d", time.Now().UnixMilli()),
		ActivityID:  current.ActivityID,
		CompletedAt: time.Now(),
		Type:        sessionType,
	}

	m.state.Logs = append([]ActivityLog{log}, m.state.Logs...)
	m.state.Queue = append(rest, QueueItem{
		InstanceID: fmt.Sprintf("q-%d", time.Now().UnixMilli()),
		ActivityID: current.ActivityID,
	})

	_ = saveState(m.filePath, m.state)
	actName := m.state.Activities[current.ActivityID].Name
	m.statusMsg = fmt.Sprintf("Concluído: %s (%s)", actName, sessionType)
}

// SkipCurrent move o item atual para a 2ª posição sem registrar falta
func (m *model) skipCurrent() {
	if len(m.state.Queue) <= 1 {
		return
	}

	first := m.state.Queue[0]
	second := m.state.Queue[1]
	rest := m.state.Queue[2:]

	m.state.Queue = append([]QueueItem{second, first}, rest...)
	_ = saveState(m.filePath, m.state)

	actName := m.state.Activities[first.ActivityID].Name
	m.statusMsg = fmt.Sprintf("Desviado para trás: %s", actName)
}

// calculateWeeklyStats calcula execuções a partir da 00:00 da segunda-feira atual
func (m model) calculateWeeklyStats() map[string]WeeklyStat {
	now := time.Now()
	weekday := int(now.Weekday())
	daysSinceMonday := (weekday + 6) % 7
	monday := time.Date(now.Year(), now.Month(), now.Day()-daysSinceMonday, 0, 0, 0, 0, now.Location())

	stats := make(map[string]WeeklyStat)
	for id := range m.state.Activities {
		stats[id] = WeeklyStat{}
	}

	for _, l := range m.state.Logs {
		if l.CompletedAt.After(monday) || l.CompletedAt.Equal(monday) {
			s := stats[l.ActivityID]
			if l.Type == "micro" {
				s.Micro++
			} else {
				s.Full++
			}
			s.Total++
			stats[l.ActivityID] = s
		}
	}
	return stats
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "r":
			if m.mode == viewDashboard {
				m.mode = viewRules
			} else {
				m.mode = viewDashboard
			}
			return m, nil

		case "esc":
			if m.mode == viewRules {
				m.mode = viewDashboard
			}
			return m, nil
		}

		if m.mode == viewDashboard {
			switch msg.String() {
			case "c", "enter":
				m.completeCurrent("full")
			case "m":
				m.completeCurrent("micro")
			case "s", "p":
				m.skipCurrent()
			}
		} else if m.mode == viewRules {
			switch msg.String() {
			case "up", "k":
				if m.ruleCursor > 0 {
					m.ruleCursor--
				}
			case "down", "j":
				if m.ruleCursor < len(m.state.Rules)-1 {
					m.ruleCursor++
				}
			}
		}
	}

	return m, nil
}

// ==========================================
// 5. Renderização de Telas
// ==========================================

func (m model) View() string {
	var b strings.Builder

	// Cabeçalho Principal
	b.WriteString(headerTitleStyle.Render("⚡ FILA DINÂMICA"))
	b.WriteString("  ")
	b.WriteString(headerSubStyle.Render("Execução contínua por prioridade"))
	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Foreground(colorMuted).Render(fmt.Sprintf("Armazenamento ativo: %s", m.filePath)))
	b.WriteString("\n\n")

	if m.mode == viewRules {
		b.WriteString(m.renderRulesView())
	} else {
		b.WriteString(m.renderDashboardView())
	}

	// Status / Notificação
	if m.statusMsg != "" {
		b.WriteString(lipgloss.NewStyle().Foreground(colorEmerald).Bold(true).Render("✔ " + m.statusMsg))
		b.WriteString("\n")
	}

	// Atalhos no rodapé
	b.WriteString("\n")
	b.WriteString(m.renderFooter())

	return b.String()
}

func (m model) renderDashboardView() string {
	var b strings.Builder

	// 1. Card da Atividade Ativa
	if len(m.state.Queue) == 0 {
		b.WriteString(boxStyle.Render("Nenhuma atividade na fila."))
		b.WriteString("\n")
	} else {
		currentQueueItem := m.state.Queue[0]
		act := m.state.Activities[currentQueueItem.ActivityID]
		catColor := getCategoryColor(act.Category)

		categoryBadge := badgeStyle.Copy().
			Background(catColor).
			Render(strings.ToUpper(string(act.Category)))

		cardContent := fmt.Sprintf(
			"%s  %s\n\n%s\n\n%s %s min/sessão  •  Meta: %dx/sem",
			lipgloss.NewStyle().Bold(true).Foreground(colorAmber).Render("● PRÓXIMO DA FILA"),
			categoryBadge,
			lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF")).Render(act.Name),
			lipgloss.NewStyle().Foreground(colorMuted).Render("Duração:"),
			lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf("%d", act.RecommendedMinutes)),
			act.TargetPerWeek,
		)
		b.WriteString(cardStyle.Render(cardContent))
		b.WriteString("\n")
	}

	// 2. Na Esteira (Próximos Itens)
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(colorMuted).Render("NA ESTEIRA (SEQUÊNCIA IMEDIATA)"))
	b.WriteString("\n")

	var upcomingTags []string
	upcomingLimit := 6
	if len(m.state.Queue) < upcomingLimit {
		upcomingLimit = len(m.state.Queue)
	}

	for i := 1; i < upcomingLimit; i++ {
		item := m.state.Queue[i]
		act := m.state.Activities[item.ActivityID]
		tag := fmt.Sprintf("+%d %s", i, act.Name)
		upcomingTags = append(upcomingTags, lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(colorBorder).
			Foreground(lipgloss.Color("#D4D4D4")).
			Padding(0, 1).
			Render(tag))
	}

	if len(upcomingTags) > 0 {
		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, upcomingTags...))
		b.WriteString("\n\n")
	} else {
		b.WriteString(headerSubStyle.Render("Fila vazia ou com apenas um item."))
		b.WriteString("\n\n")
	}

	// 3. Placar Semanal
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(colorMuted).Render("PLACAR SEMANAL (SEG - DOM)"))
	b.WriteString("\n")

	stats := m.calculateWeeklyStats()
	for _, act := range m.state.Activities {
		st := stats[act.ID]
		bar := renderProgressBar(st.Total, act.TargetPerWeek, 14)
		microTag := ""
		if st.Micro > 0 {
			microTag = lipgloss.NewStyle().Foreground(colorAmber).Render(fmt.Sprintf("(%d micro)", st.Micro))
		}

		line := fmt.Sprintf(
			"%-22s %s  %d/%dx %s",
			act.Name,
			bar,
			st.Total,
			act.TargetPerWeek,
			microTag,
		)
		b.WriteString(line + "\n")
	}
	b.WriteString("\n")

	return b.String()
}

func (m model) renderRulesView() string {
	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(colorAmber).Render("MANUAL DE CONTINGÊNCIAS & REGRAS OPERACIONAIS"))
	b.WriteString("\n\n")

	for i, rule := range m.state.Rules {
		cursor := "  "
		if i == m.ruleCursor {
			cursor = lipgloss.NewStyle().Foreground(colorAmber).Bold(true).Render("▸ ")
		}

		trigger := lipgloss.NewStyle().Foreground(colorAmber).Bold(true).Render(rule.Trigger)
		instruction := lipgloss.NewStyle().Foreground(lipgloss.Color("#E5E5E5")).Render(rule.Instruction)

		block := fmt.Sprintf("%s%s\n    %s\n", cursor, trigger, instruction)
		b.WriteString(block + "\n")
	}

	return b.String()
}

func (m model) renderFooter() string {
	if m.mode == viewRules {
		return fmt.Sprintf(
			"%s voltar • %s navegar",
			keyStyle.Render("[ESC/R]"),
			keyStyle.Render("[↑/↓]"),
		)
	}

	return fmt.Sprintf(
		"%s concluir • %s microbloco (20m) • %s desviar/pular • %s regras • %s sair",
		keyStyle.Render("[C/Enter]"),
		keyStyle.Render("[M]"),
		keyStyle.Render("[S]"),
		keyStyle.Render("[R]"),
		keyStyle.Render("[Q]"),
	)
}

func renderProgressBar(current, target, width int) string {
	if target <= 0 {
		target = 1
	}
	ratio := float64(current) / float64(target)
	if ratio > 1.0 {
		ratio = 1.0
	}
	filledLen := int(ratio * float64(width))
	if filledLen > width {
		filledLen = width
	}

	filledColor := colorAmber
	if current >= target {
		filledColor = colorEmerald
	}

	filled := lipgloss.NewStyle().Foreground(filledColor).Render(strings.Repeat("█", filledLen))
	empty := lipgloss.NewStyle().Foreground(lipgloss.Color("#3F3F46")).Render(strings.Repeat("░", width-filledLen))

	return filled + empty
}

// ==========================================
// 6. Entrypoint
// ==========================================

func main() {
	path := stateFileName
	if len(os.Args) > 1 {
		path = os.Args[1]
	}

	absPath, err := filepath.Abs(path)
	if err == nil {
		path = absPath
	}

	p := tea.NewProgram(initialModel(path), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Erro ao rodar TUI: %v\n", err)
		os.Exit(1)
	}
}
