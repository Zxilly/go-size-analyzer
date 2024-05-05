package tui

func (m *viewModel) View() string {
	return appStyle.Render(m.list.View())
}
