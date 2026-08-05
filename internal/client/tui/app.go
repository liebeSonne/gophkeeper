package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"

	apiClient "github.com/liebeSonne/gophkeeper/internal/client/adapter/gophkeeper"
	clientapp "github.com/liebeSonne/gophkeeper/internal/client/app"
	"github.com/liebeSonne/gophkeeper/internal/client/model"
	gophkeeper "github.com/liebeSonne/gophkeeper/pkg/client/gophkeeper"
)

type appView int

const (
	viewLogin appView = iota
	viewList
	viewDetail
	viewEditor
	viewFilesDetail
)

type App struct {
	view     appView
	login    LoginModel
	list     ListModel
	detail   DetailModel
	editor   EditorModel
	search   SearchModel
	client   *apiClient.Client
	token    *model.Token
	width    int
	height   int
	quit     bool
	message  string
	msgTimer *time.Timer
}

func NewApp() (*App, error) {
	a, err := clientapp.EnsureInitialized("")
	if err != nil {
		return nil, fmt.Errorf("initialize app: %w", err)
	}

	api, err := createAPIClient(a)
	if err != nil {
		return nil, fmt.Errorf("create api client: %w", err)
	}

	tokens, _ := a.Storage.GetTokens()

	return &App{
		view:   determineInitialView(tokens),
		login:  NewLoginModel(),
		list:   NewListModel(),
		detail: NewDetailModel(),
		editor: NewEditorModel(),
		search: NewSearchModel(),
		client: api,
		token:  tokens,
	}, nil
}

func determineInitialView(tokens *model.Token) appView {
	if tokens != nil && !tokens.AccessTokenExpiresAt.Before(time.Now()) {
		return viewList
	}
	return viewLogin
}

func createAPIClient(a *clientapp.App) (*apiClient.Client, error) {
	client, err := apiClient.NewClient(a.Config.ServerAddress)
	if err != nil {
		return nil, err
	}

	tokens, _ := a.Storage.GetTokens()
	if tokens != nil {
		client.SetAuthToken(tokens.AccessToken)
		client.SetRefreshToken(func(ctx context.Context) error {
			resp, err := client.RefreshTokenAPI(ctx, tokens.RefreshToken)
			if err != nil {
				return err
			}
			if resp.JSON200 != nil {
				client.SetAuthToken(resp.JSON200.AccessToken)
			}
			return nil
		})
	}

	return client, nil
}

func (a *App) Init() tea.Cmd {
	return tea.Batch(
		tea.WindowSize(),
		a.loadListData,
	)
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		_, _ = a.login.Update(msg)
		_, _ = a.list.Update(msg)
		return a, nil

	case tea.KeyMsg:
		if a.search.IsActive() {
			return a.handleSearchInput(msg)
		}

		switch msg.String() {
		case "q":
			a.quit = true
			return a, tea.Quit
		case "/":
			a.search.Activate()
			return a, nil
		}

		switch a.view {
		case viewLogin:
			return a.handleLogin(msg)
		case viewList:
			return a.handleList(msg)
		case viewDetail, viewFilesDetail:
			return a.handleDetail(msg)
		case viewEditor:
			return a.handleEditor(msg)
		}

	case *time.Timer:
		a.message = ""
	}

	switch a.view {
	case viewLogin:
		m, cmd := a.login.Update(msg)
		a.login = m.(LoginModel)
		return a, cmd
	case viewList:
		m, cmd := a.list.Update(msg)
		a.list = m.(ListModel)
		return a, cmd
	case viewDetail, viewFilesDetail:
		m, cmd := a.detail.Update(msg)
		a.detail = m.(DetailModel)
		return a, cmd
	case viewEditor:
		m, cmd := a.editor.Update(msg)
		a.editor = m.(EditorModel)
		return a, cmd
	}

	return a, nil
}

func (a *App) handleLogin(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m, cmd := a.login.Update(msg)
	a.login = m.(LoginModel)

	if a.login.IsConnecting() {
		return a, a.performLogin
	}

	if a.login.HasError() {
		a.login.Reset()
		return a, nil
	}

	return a, cmd
}

func (a *App) performLogin() tea.Msg {
	resp, err := a.client.LoginUser(context.Background(), a.login.GetLogin(), a.login.GetPassword())
	if err != nil {
		a.login.SetError(err)
		return nil
	}

	if resp.JSON200 == nil {
		a.login.SetError(fmt.Errorf("unexpected response"))
		return nil
	}

	a.token = &model.Token{
		AccessToken:          resp.JSON200.AccessToken,
		RefreshToken:         resp.JSON200.RefreshToken,
		AccessTokenExpiresAt: time.Now().Add(15 * time.Minute),
		RefreshExpiresAt:     time.Now().Add(24 * time.Hour),
	}

	a.client.SetAuthToken(a.token.AccessToken)
	a.view = viewList
	a.login.Reset()

	go a.loadListData()
	return nil
}

func (a *App) handleList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case keyEnter:
		item := a.list.GetSelectedItem()
		if item == nil {
			return a, nil
		}
		if a.list.GetTab() == 1 {
			cmd := a.loadFileDetail(item.ID)
			return a, cmd
		}
		cmd := a.loadDetail(item.ID)
		return a, cmd
	case "n":
		a.view = viewEditor
		a.editor.Reset()
		return a, nil
	case "d":
		item := a.list.GetSelectedItem()
		if item == nil {
			return a, nil
		}
		cmd := a.deleteItem(item.ID)
		return a, cmd
	case "tab", "1":
		a.list.tab = tabData
		return a, a.loadListData
	case "2":
		a.list.tab = tabFiles
		return a, a.loadListFiles
	case "g":
		a.list.page = 1
		return a, a.loadListData
	case "G":
		a.list.page = a.list.totalPages
		return a, a.loadListData
	}

	m, cmd := a.list.Update(msg)
	a.list = m.(ListModel)
	return a, cmd
}

func (a *App) loadListData() tea.Msg {
	page := a.list.GetPage()
	pageSize := 20

	var typesPtr *[]gophkeeper.DataType
	if a.list.GetFilterType() != "" {
		types := []gophkeeper.DataType{gophkeeper.DataType(a.list.GetFilterType())}
		typesPtr = &types
	}

	var queryPtr *string
	if a.list.GetSearch() != "" {
		q := a.list.GetSearch()
		queryPtr = &q
	}

	resp, err := a.client.ListData(context.Background(), &page, &pageSize, typesPtr, queryPtr)
	if err != nil {
		a.list.SetError(err)
		return nil
	}

	if resp.JSON200 == nil {
		return nil
	}

	var items []ListItem
	for _, item := range resp.JSON200.Items {
		if li := a.convertDataInfoToListItem(&item); li != nil {
			items = append(items, *li)
		}
	}

	a.list.SetItems(items, resp.JSON200.Page, resp.JSON200.TotalPages, resp.JSON200.Total)
	a.view = viewList
	return nil
}

func (a *App) loadListFiles() tea.Msg {
	page := a.list.GetPage()
	pageSize := 20

	var queryPtr *string
	if a.list.GetSearch() != "" {
		q := a.list.GetSearch()
		queryPtr = &q
	}

	resp, err := a.client.ListFiles(context.Background(), &page, &pageSize, queryPtr)
	if err != nil {
		a.list.SetError(err)
		return nil
	}

	if resp.JSON200 == nil {
		return nil
	}

	var items []ListItem
	for _, item := range resp.JSON200.Items {
		var size uint64
		if item.Size >= 0 {
			size = uint64(item.Size) // nolint:gosec
		}
		items = append(items, ListItem{
			ID:     item.Id,
			Title:  item.Name,
			Size:   humanFileSize(size),
			Status: string(item.Status),
			IsFile: true,
		})
	}

	a.list.SetItems(items, resp.JSON200.Page, resp.JSON200.TotalPages, resp.JSON200.Total)
	a.view = viewList
	return nil
}

func (a *App) convertDataInfoToListItem(item *gophkeeper.DataInfo) *ListItem {
	discriminator, err := item.Discriminator()
	if err != nil {
		return nil
	}

	var title, dtype string
	var created time.Time
	var id uuid.UUID

	switch discriminator {
	case dataTypeLoginPassword:
		if data, err := item.AsLoginPasswordDataInfo(); err == nil {
			id = data.Id
			title = data.Login
			dtype = dataTypeLoginPassword
			created = data.CreatedAt
		}
	case dataTypeBankCard:
		if data, err := item.AsBankCardDataInfo(); err == nil {
			id = data.Id
			title = data.CardHolder
			dtype = dataTypeBankCard
			created = data.CreatedAt
		}
	case dataTypeText:
		if data, err := item.AsTextDataInfo(); err == nil {
			id = data.Id
			title = truncateString(data.Text, 20)
			dtype = dataTypeText
			created = data.CreatedAt
		}
	case dataTypeFile:
		if data, err := item.AsFileDataInfo(); err == nil {
			id = data.Id
			title = fmt.Sprintf("%d files", len(data.FileIds))
			dtype = dataTypeFile
			created = data.CreatedAt
		}
	}

	return &ListItem{
		ID:      id,
		Title:   title,
		Type:    dtype,
		Created: created,
	}
}

func (a *App) loadDetail(id uuid.UUID) tea.Cmd {
	return func() tea.Msg {
		resp, err := a.client.GetData(context.Background(), id)
		if err != nil {
			a.showMessage("Error: " + err.Error())
			return nil
		}

		if resp.JSON200 == nil {
			return nil
		}

		a.detail = a.convertDataInfoToDetail(resp.JSON200)
		a.view = viewDetail
		return nil
	}
}

func (a *App) convertDataInfoToDetail(info *gophkeeper.DataInfo) DetailModel {
	discriminator, _ := info.Discriminator()

	switch discriminator {
	case dataTypeLoginPassword:
		if data, err := info.AsLoginPasswordDataInfo(); err == nil {
			return NewLoginPasswordDetail(data.Id, data.Login, data.Password, data.CreatedAt, data.UpdatedAt)
		}
	case dataTypeBankCard:
		if data, err := info.AsBankCardDataInfo(); err == nil {
			return NewBankCardDetail(data.Id, data.CardNumber, data.CardHolder, data.CardExpiry, data.CardCvv, data.CreatedAt, data.UpdatedAt)
		}
	case dataTypeText:
		if data, err := info.AsTextDataInfo(); err == nil {
			return NewTextDetail(data.Id, data.Text, data.CreatedAt, data.UpdatedAt)
		}
	case dataTypeFile:
		if data, err := info.AsFileDataInfo(); err == nil {
			return NewFileDetail(data.Id, data.FileIds, data.CreatedAt, data.UpdatedAt)
		}
	}

	return NewDetailModel()
}

func (a *App) loadFileDetail(id uuid.UUID) tea.Cmd {
	return func() tea.Msg {
		a.detail = NewFileItemDetail(id, "", "", "", 0, time.Time{}, time.Time{})
		a.view = viewFilesDetail
		return nil
	}
}

func (a *App) deleteItem(id uuid.UUID) tea.Cmd {
	return func() tea.Msg {
		if err := a.client.DeleteData(context.Background(), id); err != nil {
			a.showMessage("Delete failed: " + err.Error())
			return nil
		}
		a.showMessage("Deleted")
		a.loadListData()
		return nil
	}
}

func (a *App) handleDetail(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case keyEsc:
		a.view = viewList
		return a, nil
	case "e":
		a.view = viewEditor
		a.editor.Reset()
		return a, nil
	case "d":
		if a.detail.ID != uuid.Nil {
			cmd := a.deleteItem(a.detail.ID)
			return a, cmd
		}
	}

	m, cmd := a.detail.Update(msg)
	a.detail = m.(DetailModel)
	return a, cmd
}

func (a *App) handleEditor(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m, cmd := a.editor.Update(msg)
	a.editor = m.(EditorModel)

	if a.editor.IsDone() {
		a.view = viewList
		return a, a.loadListData
	}

	if msg.String() == keyEsc && a.editor.state == editorTypeSelect {
		a.view = viewList
		return a, nil
	}

	return a, cmd
}

func (a *App) handleSearchInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m, cmd := a.search.Update(msg)
	a.search = m.(SearchModel)

	if !a.search.IsActive() && a.search.GetQuery() != "" {
		a.list.SetSearch(a.search.GetQuery())
		a.list.page = 1
		if a.list.GetTab() == 0 {
			return a, a.loadListData
		}
		return a, a.loadListFiles
	}

	return a, cmd
}

func (a *App) View() string {
	if a.quit {
		return ""
	}

	var sb strings.Builder

	switch a.view {
	case viewLogin:
		sb.WriteString(a.login.View())
	case viewList:
		sb.WriteString(a.list.View())
	case viewDetail, viewFilesDetail:
		sb.WriteString(a.detail.View())
	case viewEditor:
		sb.WriteString(a.editor.View())
	}

	if a.search.IsActive() {
		sb.WriteString(a.search.View())
	}

	sb.WriteString("\n")
	sb.WriteString(a.renderStatusBar())

	if a.message != "" {
		sb.WriteString("\n")
		sb.WriteString(styleActive.Render("  " + a.message + "\n"))
	}

	return sb.String()
}

func (a *App) renderStatusBar() string {
	var parts []string

	if a.token != nil {
		parts = append(parts, styleBold.Render("authenticated"))
	} else {
		parts = append(parts, styleFaint.Render("not logged in"))
	}

	if a.view == viewList {
		tab := "Data"
		if a.list.GetTab() == 1 {
			tab = "Files"
		}
		parts = append(parts, tab)
		parts = append(parts, fmt.Sprintf("%d items", a.list.total))
	}

	help := a.getStatusBarHelp()
	if help != "" {
		parts = append(parts, help)
	}

	return styleStatusBar.Render(strings.Join(parts, separator()))
}

func (a *App) getStatusBarHelp() string {
	switch a.view {
	case viewLogin:
		return "Tab: next  Enter: connect  Esc: quit"
	case viewList:
		return "j/k: nav  Enter: open  n: new  d: del  /: search  q: quit"
	case viewDetail, viewFilesDetail:
		return "j/k: field  c: copy  e: edit  d: del  esc: back"
	case viewEditor:
		return "j/k: nav  Enter: next  Esc: back"
	default:
		return "q: quit"
	}
}

func (a *App) showMessage(msg string) {
	a.message = msg
	if a.msgTimer != nil {
		a.msgTimer.Stop()
	}
	a.msgTimer = time.NewTimer(3 * time.Second)
}

func truncateString(s string, maxLen int) string {
	r := []rune(s)
	if len(r) <= maxLen {
		return s
	}
	return string(r[:maxLen]) + "..."
}

func humanFileSize(size uint64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := uint64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}
