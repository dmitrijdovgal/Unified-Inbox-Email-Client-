// mail.go
// Unified Inbox на Go

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"
)

// ANSI-цвета
const (
	reset  = "\033[0m"
	green  = "\033[92m"
	yellow = "\033[93m"
	cyan   = "\033[96m"
	red    = "\033[91m"
	bold   = "\033[1m"
)

func colorize(text, color string) string {
	return color + text + reset
}

type Message struct {
	ID      int    `json:"id"`
	From    string `json:"from"`
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
	Date    string `json:"date"`
	Read    bool   `json:"read"`
}

type Account struct {
	Name     string    `json:"name"`
	Email    string    `json:"email"`
	Messages []Message `json:"messages"`
	Sent     []Message `json:"sent"`
	Trash    []Message `json:"trash"`
	NextID   int       `json:"next_id"`
}

type Data struct {
	Accounts []Account `json:"accounts"`
	Current  string    `json:"current"`
}

type MailClient struct {
	dataFile string
	data     Data
	current  string
}

func NewMailClient(dataFile string) *MailClient {
	mc := &MailClient{dataFile: dataFile}
	mc.load()
	return mc
}

func (m *MailClient) load() {
	file, err := ioutil.ReadFile(m.dataFile)
	if err != nil {
		m.data = Data{Accounts: []Account{}, Current: ""}
		return
	}
	var d Data
	if err := json.Unmarshal(file, &d); err != nil {
		m.data = Data{Accounts: []Account{}, Current: ""}
	} else {
		m.data = d
		m.current = d.Current
	}
}

func (m *MailClient) save() {
	data, _ := json.MarshalIndent(m.data, "", "  ")
	ioutil.WriteFile(m.dataFile, data, 0644)
}

func (m *MailClient) getAccount(email string) *Account {
	for i := range m.data.Accounts {
		if m.data.Accounts[i].Email == email {
			return &m.data.Accounts[i]
		}
	}
	return nil
}

func (m *MailClient) getCurrentAccount() *Account {
	if m.current == "" {
		return nil
	}
	return m.getAccount(m.current)
}

func (m *MailClient) AddAccount(name, email string) bool {
	if m.getAccount(email) != nil {
		return false
	}
	acc := Account{
		Name:     name,
		Email:    email,
		Messages: []Message{},
		Sent:     []Message{},
		Trash:    []Message{},
		NextID:   1,
	}
	m.data.Accounts = append(m.data.Accounts, acc)
	if m.current == "" {
		m.data.Current = email
		m.current = email
	}
	m.save()
	return true
}

func (m *MailClient) SwitchAccount(email string) bool {
	if m.getAccount(email) == nil {
		return false
	}
	m.data.Current = email
	m.current = email
	m.save()
	return true
}

func (m *MailClient) ListMessages() []Message {
	acc := m.getCurrentAccount()
	if acc == nil {
		return []Message{}
	}
	all := append(acc.Messages, acc.Sent...)
	all = append(all, acc.Trash...)
	// sort by date descending
	for i := 0; i < len(all); i++ {
		for j := i + 1; j < len(all); j++ {
			if all[i].Date < all[j].Date {
				all[i], all[j] = all[j], all[i]
			}
		}
	}
	return all
}

func (m *MailClient) GetMessage(id int) *Message {
	acc := m.getCurrentAccount()
	if acc == nil {
		return nil
	}
	for _, list := range [][]Message{acc.Messages, acc.Sent, acc.Trash} {
		for i := range list {
			if list[i].ID == id {
				list[i].Read = true
				m.save()
				return &list[i]
			}
		}
	}
	return nil
}

func (m *MailClient) SendMessage(to, subject, body string) bool {
	acc := m.getCurrentAccount()
	if acc == nil {
		return false
	}
	msg := Message{
		ID:      acc.NextID,
		From:    acc.Email,
		To:      to,
		Subject: subject,
		Body:    body,
		Date:    time.Now().Format(time.RFC3339),
		Read:    true,
	}
	acc.NextID++
	acc.Sent = append(acc.Sent, msg)
	m.save()
	return true
}

func (m *MailClient) DeleteMessage(id int) bool {
	acc := m.getCurrentAccount()
	if acc == nil {
		return false
	}
	for i := range acc.Messages {
		if acc.Messages[i].ID == id {
			acc.Trash = append(acc.Trash, acc.Messages[i])
			acc.Messages = append(acc.Messages[:i], acc.Messages[i+1:]...)
			m.save()
			return true
		}
	}
	for i := range acc.Sent {
		if acc.Sent[i].ID == id {
			acc.Trash = append(acc.Trash, acc.Sent[i])
			acc.Sent = append(acc.Sent[:i], acc.Sent[i+1:]...)
			m.save()
			return true
		}
	}
	return false
}

func (m *MailClient) Search(query string) []Message {
	acc := m.getCurrentAccount()
	if acc == nil {
		return []Message{}
	}
	q := strings.ToLower(query)
	results := []Message{}
	for _, list := range [][]Message{acc.Messages, acc.Sent, acc.Trash} {
		for _, msg := range list {
			if strings.Contains(strings.ToLower(msg.Subject), q) ||
				strings.Contains(strings.ToLower(msg.Body), q) ||
				strings.Contains(strings.ToLower(msg.From), q) ||
				strings.Contains(strings.ToLower(msg.To), q) {
				results = append(results, msg)
			}
		}
	}
	for i := 0; i < len(results); i++ {
		for j := i + 1; j < len(results); j++ {
			if results[i].Date < results[j].Date {
				results[i], results[j] = results[j], results[i]
			}
		}
	}
	return results
}

func (m *MailClient) Export(filename string) bool {
	acc := m.getCurrentAccount()
	if acc == nil {
		return false
	}
	all := append(acc.Messages, acc.Sent...)
	all = append(all, acc.Trash...)
	content := ""
	for _, msg := range all {
		content += fmt.Sprintf("From %s %s\nTo: %s\nSubject: %s\n\n%s\n\n",
			msg.From, msg.Date, msg.To, msg.Subject, msg.Body)
	}
	ioutil.WriteFile(filename, []byte(content), 0644)
	return true
}

func (m *MailClient) ListAccounts() []Account {
	return m.data.Accounts
}

func (m *MailClient) GetCurrentEmail() string {
	return m.current
}

func main() {
	if len(os.Args) < 2 || os.Args[1] == "help" {
		fmt.Println(`Использование: go run mail.go <команда> [опции]
  add-account   --name <name> --email <email>
  switch        --email <email>
  list
  read          --id <id>
  send          --to <to> --subject <subject> --body <body>
  delete        --id <id>
  search        --query <query>
  export        [--file <file>]
  accounts
  help`)
		os.Exit(0)
	}
	cmd := os.Args[1]
	opts := make(map[string]string)
	for i := 2; i < len(os.Args); i++ {
		arg := os.Args[i]
		if strings.HasPrefix(arg, "--") {
			key := arg[2:]
			if i+1 < len(os.Args) && !strings.HasPrefix(os.Args[i+1], "--") {
				opts[key] = os.Args[i+1]
				i++
			} else {
				opts[key] = ""
			}
		}
	}
	dataFile := opts["data"]
	if dataFile == "" {
		dataFile = "mailbox.json"
	}
	client := NewMailClient(dataFile)

	switch cmd {
	case "add-account":
		name, ok1 := opts["name"]
		email, ok2 := opts["email"]
		if !ok1 || !ok2 {
			fmt.Println("Требуются --name и --email")
			os.Exit(1)
		}
		if client.AddAccount(name, email) {
			fmt.Printf("Аккаунт %s добавлен.\n", email)
		} else {
			fmt.Printf("Аккаунт %s уже существует.\n", email)
		}
	case "switch":
		email, ok := opts["email"]
		if !ok {
			fmt.Println("Требуется --email")
			os.Exit(1)
		}
		if client.SwitchAccount(email) {
			fmt.Printf("Переключено на %s\n", email)
		} else {
			fmt.Printf("Аккаунт %s не найден.\n", email)
		}
	case "list":
		msgs := client.ListMessages()
		if len(msgs) == 0 {
			fmt.Println("Нет писем.")
		} else {
			current := client.GetCurrentEmail()
			fmt.Println(colorize("Почта: "+current, bold+cyan))
			for _, msg := range msgs {
				status := "🔵"
				if !msg.Read {
					status = "⚪"
				}
				fmt.Printf("%s %s | %s | %s | %s\n",
					status,
					colorize(strconv.Itoa(msg.ID), yellow),
					colorize(msg.From, green),
					colorize(msg.Subject, cyan),
					msg.Date[:10])
			}
		}
	case "read":
		idStr, ok := opts["id"]
		if !ok {
			fmt.Println("Требуется --id")
			os.Exit(1)
		}
		id, _ := strconv.Atoi(idStr)
		msg := client.GetMessage(id)
		if msg == nil {
			fmt.Println("Письмо не найдено.")
		} else {
			fmt.Println(colorize("От: "+msg.From, green))
			fmt.Println(colorize("Кому: "+msg.To, green))
			fmt.Println(colorize("Тема: "+msg.Subject, cyan))
			fmt.Println(colorize("Дата: "+msg.Date, yellow))
			fmt.Println("\n" + msg.Body)
		}
	case "send":
		to, ok1 := opts["to"]
		subject, ok2 := opts["subject"]
		body, ok3 := opts["body"]
		if !ok1 || !ok2 || !ok3 {
			fmt.Println("Требуются --to, --subject, --body")
			os.Exit(1)
		}
		if client.SendMessage(to, subject, body) {
			fmt.Println("Письмо отправлено.")
		} else {
			fmt.Println("Ошибка отправки (нет активного аккаунта).")
		}
	case "delete":
		idStr, ok := opts["id"]
		if !ok {
			fmt.Println("Требуется --id")
			os.Exit(1)
		}
		id, _ := strconv.Atoi(idStr)
		if client.DeleteMessage(id) {
			fmt.Printf("Письмо %d удалено.\n", id)
		} else {
			fmt.Println("Письмо не найдено.")
		}
	case "search":
		query, ok := opts["query"]
		if !ok {
			fmt.Println("Требуется --query")
			os.Exit(1)
		}
		results := client.Search(query)
		if len(results) == 0 {
			fmt.Println("Ничего не найдено.")
		} else {
			for _, msg := range results {
				status := "🔵"
				if !msg.Read {
					status = "⚪"
				}
				fmt.Printf("%s %s | %s | %s | %s\n",
					status,
					colorize(strconv.Itoa(msg.ID), yellow),
					colorize(msg.From, green),
					colorize(msg.Subject, cyan),
					msg.Date[:10])
			}
		}
	case "export":
		filename := opts["file"]
		if filename == "" {
			filename = "export.mbox"
		}
		if client.Export(filename) {
			fmt.Printf("Экспорт завершён в %s\n", filename)
		} else {
			fmt.Println("Ошибка экспорта (нет активного аккаунта).")
		}
	case "accounts":
		accs := client.ListAccounts()
		current := client.GetCurrentEmail()
		if len(accs) == 0 {
			fmt.Println("Нет аккаунтов.")
		} else {
			for _, acc := range accs {
				marker := ""
				if acc.Email == current {
					marker = " *"
				}
				fmt.Printf("%s (%s)%s\n", acc.Email, acc.Name, marker)
			}
		}
	default:
		fmt.Println("Неизвестная команда.")
	}
}
