// MailClient.cs
// Unified Inbox на C#

using System;
using System.Collections.Generic;
using System.IO;
using System.Linq;
using System.Text;
using System.Text.Json;

class MailClient
{
    private const string RESET = "\u001B[0m";
    private const string GREEN = "\u001B[92m";
    private const string YELLOW = "\u001B[93m";
    private const string CYAN = "\u001B[96m";
    private const string RED = "\u001B[91m";
    private const string BOLD = "\u001B[1m";

    private static string Colorize(string text, string color) => color + text + RESET;

    private class Message
    {
        public int Id { get; set; }
        public string From { get; set; }
        public string To { get; set; }
        public string Subject { get; set; }
        public string Body { get; set; }
        public string Date { get; set; }
        public bool Read { get; set; }
    }

    private class Account
    {
        public string Name { get; set; }
        public string Email { get; set; }
        public List<Message> Messages { get; set; } = new();
        public List<Message> Sent { get; set; } = new();
        public List<Message> Trash { get; set; } = new();
        public int NextId { get; set; } = 1;
    }

    private class Data
    {
        public List<Account> Accounts { get; set; } = new();
        public string Current { get; set; }
    }

    private string dataFile;
    private Data data = new();
    private string currentEmail;

    public MailClient(string dataFile)
    {
        this.dataFile = dataFile;
        Load();
    }

    private void Load()
    {
        if (!File.Exists(dataFile)) return;
        try
        {
            string json = File.ReadAllText(dataFile);
            data = JsonSerializer.Deserialize<Data>(json) ?? new Data();
            currentEmail = data.Current;
        }
        catch { data = new Data(); }
    }

    private void Save()
    {
        string json = JsonSerializer.Serialize(data, new JsonSerializerOptions { WriteIndented = true });
        File.WriteAllText(dataFile, json);
    }

    private Account GetAccount(string email) => data.Accounts.FirstOrDefault(a => a.Email == email);
    private Account GetCurrentAccount() => currentEmail == null ? null : GetAccount(currentEmail);

    public bool AddAccount(string name, string email)
    {
        if (GetAccount(email) != null) return false;
        var acc = new Account { Name = name, Email = email };
        data.Accounts.Add(acc);
        if (currentEmail == null)
        {
            data.Current = email;
            currentEmail = email;
        }
        Save();
        return true;
    }

    public bool SwitchAccount(string email)
    {
        if (GetAccount(email) == null) return false;
        data.Current = email;
        currentEmail = email;
        Save();
        return true;
    }

    public List<Message> ListMessages()
    {
        var acc = GetCurrentAccount();
        if (acc == null) return new List<Message>();
        var all = acc.Messages.Concat(acc.Sent).Concat(acc.Trash).ToList();
        all.Sort((a,b) => string.Compare(b.Date, a.Date));
        return all;
    }

    public Message GetMessage(int id)
    {
        var acc = GetCurrentAccount();
        if (acc == null) return null;
        foreach (var list in new[] { acc.Messages, acc.Sent, acc.Trash })
        {
            var msg = list.FirstOrDefault(m => m.Id == id);
            if (msg != null)
            {
                msg.Read = true;
                Save();
                return msg;
            }
        }
        return null;
    }

    public bool SendMessage(string to, string subject, string body)
    {
        var acc = GetCurrentAccount();
        if (acc == null) return false;
        var msg = new Message
        {
            Id = acc.NextId++,
            From = acc.Email,
            To = to,
            Subject = subject,
            Body = body,
            Date = DateTime.Now.ToString("yyyy-MM-ddTHH:mm:ss"),
            Read = true
        };
        acc.Sent.Add(msg);
        Save();
        return true;
    }

    public bool DeleteMessage(int id)
    {
        var acc = GetCurrentAccount();
        if (acc == null) return false;
        foreach (var list in new[] { acc.Messages, acc.Sent })
        {
            var msg = list.FirstOrDefault(m => m.Id == id);
            if (msg != null)
            {
                list.Remove(msg);
                acc.Trash.Add(msg);
                Save();
                return true;
            }
        }
        return false;
    }

    public List<Message> Search(string query)
    {
        var acc = GetCurrentAccount();
        if (acc == null) return new List<Message>();
        var q = query.ToLower();
        var results = new List<Message>();
        foreach (var list in new[] { acc.Messages, acc.Sent, acc.Trash })
        {
            results.AddRange(list.Where(m =>
                m.Subject.ToLower().Contains(q) ||
                m.Body.ToLower().Contains(q) ||
                m.From.ToLower().Contains(q) ||
                m.To.ToLower().Contains(q)));
        }
        results.Sort((a,b) => string.Compare(b.Date, a.Date));
        return results;
    }

    public bool Export(string filename)
    {
        var acc = GetCurrentAccount();
        if (acc == null) return false;
        var all = acc.Messages.Concat(acc.Sent).Concat(acc.Trash).ToList();
        var sb = new StringBuilder();
        foreach (var m in all)
        {
            sb.Append($"From {m.From} {m.Date}\n");
            sb.Append($"To: {m.To}\n");
            sb.Append($"Subject: {m.Subject}\n");
            sb.Append($"\n{m.Body}\n\n");
        }
        File.WriteAllText(filename, sb.ToString());
        return true;
    }

    public List<Account> ListAccounts() => data.Accounts;
    public string GetCurrentEmail() => currentEmail;

    static void Main(string[] args)
    {
        if (args.Length == 0 || args[0] == "help")
        {
            Console.WriteLine(@"Использование: MailClient <команда> [опции]
  add-account   --name <name> --email <email>
  switch        --email <email>
  list
  read          --id <id>
  send          --to <to> --subject <subject> --body <body>
  delete        --id <id>
  search        --query <query>
  export        [--file <file>]
  accounts
  help");
            return;
        }
        var cmd = args[0];
        var opts = new Dictionary<string, string>();
        for (int i = 1; i < args.Length; i++)
        {
            if (args[i].StartsWith("--"))
            {
                var key = args[i].Substring(2);
                if (i+1 < args.Length && !args[i+1].StartsWith("--"))
                {
                    opts[key] = args[++i];
                }
                else opts[key] = "";
            }
        }
        var dataFile = opts.GetValueOrDefault("data", "mailbox.json");
        var client = new MailClient(dataFile);

        switch (cmd)
        {
            case "add-account":
                if (!opts.ContainsKey("name") || !opts.ContainsKey("email"))
                {
                    Console.WriteLine("Требуются --name и --email");
                    return;
                }
                if (client.AddAccount(opts["name"], opts["email"]))
                    Console.WriteLine($"Аккаунт {opts["email"]} добавлен.");
                else
                    Console.WriteLine($"Аккаунт {opts["email"]} уже существует.");
                break;
            case "switch":
                if (!opts.ContainsKey("email"))
                {
                    Console.WriteLine("Требуется --email");
                    return;
                }
                if (client.SwitchAccount(opts["email"]))
                    Console.WriteLine($"Переключено на {opts["email"]}");
                else
                    Console.WriteLine($"Аккаунт {opts["email"]} не найден.");
                break;
            case "list":
                var msgs = client.ListMessages();
                if (msgs.Count == 0) Console.WriteLine("Нет писем.");
                else
                {
                    var current = client.GetCurrentEmail();
                    Console.WriteLine(Colorize($"Почта: {current}", BOLD + CYAN));
                    foreach (var m in msgs)
                    {
                        var status = m.Read ? "🔵" : "⚪";
                        Console.WriteLine($"{status} {Colorize(m.Id.ToString(), YELLOW)} | {Colorize(m.From, GREEN)} | {Colorize(m.Subject, CYAN)} | {m.Date.Substring(0,10)}");
                    }
                }
                break;
            case "read":
                if (!opts.ContainsKey("id"))
                {
                    Console.WriteLine("Требуется --id");
                    return;
                }
                int id = int.Parse(opts["id"]);
                var msg = client.GetMessage(id);
                if (msg == null) Console.WriteLine("Письмо не найдено.");
                else
                {
                    Console.WriteLine(Colorize($"От: {msg.From}", GREEN));
                    Console.WriteLine(Colorize($"Кому: {msg.To}", GREEN));
                    Console.WriteLine(Colorize($"Тема: {msg.Subject}", CYAN));
                    Console.WriteLine(Colorize($"Дата: {msg.Date}", YELLOW));
                    Console.WriteLine("\n" + msg.Body);
                }
                break;
            case "send":
                if (!opts.ContainsKey("to") || !opts.ContainsKey("subject") || !opts.ContainsKey("body"))
                {
                    Console.WriteLine("Требуются --to, --subject, --body");
                    return;
                }
                if (client.SendMessage(opts["to"], opts["subject"], opts["body"]))
                    Console.WriteLine("Письмо отправлено.");
                else
                    Console.WriteLine("Ошибка отправки (нет активного аккаунта).");
                break;
            case "delete":
                if (!opts.ContainsKey("id"))
                {
                    Console.WriteLine("Требуется --id");
                    return;
                }
                int delId = int.Parse(opts["id"]);
                if (client.DeleteMessage(delId))
                    Console.WriteLine($"Письмо {delId} удалено.");
                else
                    Console.WriteLine("Письмо не найдено.");
                break;
            case "search":
                if (!opts.ContainsKey("query"))
                {
                    Console.WriteLine("Требуется --query");
                    return;
                }
                var results = client.Search(opts["query"]);
                if (results.Count == 0) Console.WriteLine("Ничего не найдено.");
                else
                {
                    foreach (var m in results)
                    {
                        var status = m.Read ? "🔵" : "⚪";
                        Console.WriteLine($"{status} {Colorize(m.Id.ToString(), YELLOW)} | {Colorize(m.From, GREEN)} | {Colorize(m.Subject, CYAN)} | {m.Date.Substring(0,10)}");
                    }
                }
                break;
            case "export":
                string filename = opts.GetValueOrDefault("file", "export.mbox");
                if (client.Export(filename))
                    Console.WriteLine($"Экспорт завершён в {filename}");
                else
                    Console.WriteLine("Ошибка экспорта (нет активного аккаунта).");
                break;
            case "accounts":
                var accs = client.ListAccounts();
                var cur = client.GetCurrentEmail();
                if (accs.Count == 0) Console.WriteLine("Нет аккаунтов.");
                else
                {
                    foreach (var a in accs)
                    {
                        string marker = a.Email == cur ? " *" : "";
                        Console.WriteLine($"{a.Email} ({a.Name}){marker}");
                    }
                }
                break;
            default:
                Console.WriteLine("Неизвестная команда.");
                break;
        }
    }
}
