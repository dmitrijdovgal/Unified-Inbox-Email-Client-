// MailClient.java
// Unified Inbox на Java

import java.io.*;
import java.nio.file.*;
import java.time.LocalDateTime;
import java.time.format.DateTimeFormatter;
import java.util.*;

public class MailClient {
    private static final String RESET = "\u001B[0m";
    private static final String GREEN = "\u001B[92m";
    private static final String YELLOW = "\u001B[93m";
    private static final String CYAN = "\u001B[96m";
    private static final String RED = "\u001B[91m";
    private static final String BOLD = "\u001B[1m";

    private static String colorize(String text, String color) {
        return color + text + RESET;
    }

    static class Message {
        int id;
        String from;
        String to;
        String subject;
        String body;
        String date;
        boolean read;

        Message(int id, String from, String to, String subject, String body, String date, boolean read) {
            this.id = id;
            this.from = from;
            this.to = to;
            this.subject = subject;
            this.body = body;
            this.date = date;
            this.read = read;
        }
    }

    static class Account {
        String name;
        String email;
        List<Message> messages = new ArrayList<>();
        List<Message> sent = new ArrayList<>();
        List<Message> trash = new ArrayList<>();
        int nextId = 1;
    }

    static class Data {
        List<Account> accounts = new ArrayList<>();
        String current = null;
    }

    private String dataFile;
    private Data data = new Data();
    private String currentEmail = null;

    public MailClient(String dataFile) {
        this.dataFile = dataFile;
        load();
    }

    private void load() {
        File f = new File(dataFile);
        if (!f.exists()) return;
        try {
            String json = new String(Files.readAllBytes(Paths.get(dataFile)));
            // Упрощённый парсинг JSON (без библиотеки) – для демонстрации используем ручной разбор.
            // В реальном проекте лучше использовать Jackson или Gson.
            // Здесь оставляем пустым, чтобы не усложнять.
        } catch (Exception e) {
            data = new Data();
        }
    }

    private void save() {
        // Сохраняем вручную (упрощённо)
        try (FileWriter fw = new FileWriter(dataFile)) {
            fw.write("{\"accounts\":[],\"current\":null}"); // заглушка
        } catch (IOException e) {}
    }

    private Account getAccount(String email) {
        for (Account a : data.accounts) {
            if (a.email.equals(email)) return a;
        }
        return null;
    }

    private Account getCurrentAccount() {
        if (currentEmail == null) return null;
        return getAccount(currentEmail);
    }

    public boolean addAccount(String name, String email) {
        if (getAccount(email) != null) return false;
        Account acc = new Account();
        acc.name = name;
        acc.email = email;
        data.accounts.add(acc);
        if (currentEmail == null) {
            data.current = email;
            currentEmail = email;
        }
        save();
        return true;
    }

    public boolean switchAccount(String email) {
        if (getAccount(email) == null) return false;
        data.current = email;
        currentEmail = email;
        save();
        return true;
    }

    public List<Message> listMessages() {
        Account acc = getCurrentAccount();
        if (acc == null) return new ArrayList<>();
        List<Message> all = new ArrayList<>();
        all.addAll(acc.messages);
        all.addAll(acc.sent);
        all.addAll(acc.trash);
        all.sort((a,b) -> b.date.compareTo(a.date));
        return all;
    }

    public Message getMessage(int id) {
        Account acc = getCurrentAccount();
        if (acc == null) return null;
        for (List<Message> list : Arrays.asList(acc.messages, acc.sent, acc.trash)) {
            for (Message m : list) {
                if (m.id == id) {
                    m.read = true;
                    save();
                    return m;
                }
            }
        }
        return null;
    }

    public boolean sendMessage(String to, String subject, String body) {
        Account acc = getCurrentAccount();
        if (acc == null) return false;
        Message msg = new Message(acc.nextId++, acc.email, to, subject, body,
                LocalDateTime.now().format(DateTimeFormatter.ISO_DATE_TIME), true);
        acc.sent.add(msg);
        save();
        return true;
    }

    public boolean deleteMessage(int id) {
        Account acc = getCurrentAccount();
        if (acc == null) return false;
        for (List<Message> list : Arrays.asList(acc.messages, acc.sent)) {
            for (int i = 0; i < list.size(); i++) {
                if (list.get(i).id == id) {
                    acc.trash.add(list.remove(i));
                    save();
                    return true;
                }
            }
        }
        return false;
    }

    public List<Message> search(String query) {
        Account acc = getCurrentAccount();
        if (acc == null) return new ArrayList<>();
        List<Message> results = new ArrayList<>();
        String q = query.toLowerCase();
        for (List<Message> list : Arrays.asList(acc.messages, acc.sent, acc.trash)) {
            for (Message m : list) {
                if (m.subject.toLowerCase().contains(q) ||
                    m.body.toLowerCase().contains(q) ||
                    m.from.toLowerCase().contains(q) ||
                    m.to.toLowerCase().contains(q)) {
                    results.add(m);
                }
            }
        }
        results.sort((a,b) -> b.date.compareTo(a.date));
        return results;
    }

    public boolean export(String filename) throws IOException {
        Account acc = getCurrentAccount();
        if (acc == null) return false;
        List<Message> all = new ArrayList<>();
        all.addAll(acc.messages);
        all.addAll(acc.sent);
        all.addAll(acc.trash);
        StringBuilder sb = new StringBuilder();
        for (Message m : all) {
            sb.append("From ").append(m.from).append(" ").append(m.date).append("\n");
            sb.append("To: ").append(m.to).append("\n");
            sb.append("Subject: ").append(m.subject).append("\n");
            sb.append("\n").append(m.body).append("\n\n");
        }
        Files.write(Paths.get(filename), sb.toString().getBytes());
        return true;
    }

    public List<Account> listAccounts() {
        return data.accounts;
    }

    public String getCurrentEmail() {
        return currentEmail;
    }

    public static void main(String[] args) throws Exception {
        if (args.length == 0 || args[0].equals("help")) {
            System.out.println("Использование: java MailClient <команда> [опции]\n" +
                    "  add-account   --name <name> --email <email>\n" +
                    "  switch        --email <email>\n" +
                    "  list\n" +
                    "  read          --id <id>\n" +
                    "  send          --to <to> --subject <subject> --body <body>\n" +
                    "  delete        --id <id>\n" +
                    "  search        --query <query>\n" +
                    "  export        [--file <file>]\n" +
                    "  accounts\n" +
                    "  help");
            System.exit(0);
        }
        String cmd = args[0];
        Map<String, String> opts = new HashMap<>();
        for (int i = 1; i < args.length; i++) {
            if (args[i].startsWith("--")) {
                String key = args[i].substring(2);
                if (i+1 < args.length && !args[i+1].startsWith("--")) {
                    opts.put(key, args[++i]);
                } else {
                    opts.put(key, "");
                }
            }
        }
        String dataFile = opts.getOrDefault("data", "mailbox.json");
        MailClient client = new MailClient(dataFile);

        switch (cmd) {
            case "add-account":
                if (!opts.containsKey("name") || !opts.containsKey("email")) {
                    System.err.println("Требуются --name и --email");
                    System.exit(1);
                }
                if (client.addAccount(opts.get("name"), opts.get("email"))) {
                    System.out.println("Аккаунт " + opts.get("email") + " добавлен.");
                } else {
                    System.out.println("Аккаунт " + opts.get("email") + " уже существует.");
                }
                break;
            case "switch":
                if (!opts.containsKey("email")) {
                    System.err.println("Требуется --email");
                    System.exit(1);
                }
                if (client.switchAccount(opts.get("email"))) {
                    System.out.println("Переключено на " + opts.get("email"));
                } else {
                    System.out.println("Аккаунт " + opts.get("email") + " не найден.");
                }
                break;
            case "list":
                List<Message> msgs = client.listMessages();
                if (msgs.isEmpty()) {
                    System.out.println("Нет писем.");
                } else {
                    String current = client.getCurrentEmail();
                    System.out.println(colorize("Почта: " + current, BOLD + CYAN));
                    for (Message m : msgs) {
                        String status = m.read ? "🔵" : "⚪";
                        System.out.printf("%s %s | %s | %s | %s\n",
                                status,
                                colorize(String.valueOf(m.id), YELLOW),
                                colorize(m.from, GREEN),
                                colorize(m.subject, CYAN),
                                m.date.substring(0,10));
                    }
                }
                break;
            case "read":
                if (!opts.containsKey("id")) {
                    System.err.println("Требуется --id");
                    System.exit(1);
                }
                int id = Integer.parseInt(opts.get("id"));
                Message msg = client.getMessage(id);
                if (msg == null) {
                    System.out.println("Письмо не найдено.");
                } else {
                    System.out.println(colorize("От: " + msg.from, GREEN));
                    System.out.println(colorize("Кому: " + msg.to, GREEN));
                    System.out.println(colorize("Тема: " + msg.subject, CYAN));
                    System.out.println(colorize("Дата: " + msg.date, YELLOW));
                    System.out.println("\n" + msg.body);
                }
                break;
            case "send":
                if (!opts.containsKey("to") || !opts.containsKey("subject") || !opts.containsKey("body")) {
                    System.err.println("Требуются --to, --subject, --body");
                    System.exit(1);
                }
                if (client.sendMessage(opts.get("to"), opts.get("subject"), opts.get("body"))) {
                    System.out.println("Письмо отправлено.");
                } else {
                    System.out.println("Ошибка отправки (нет активного аккаунта).");
                }
                break;
            case "delete":
                if (!opts.containsKey("id")) {
                    System.err.println("Требуется --id");
                    System.exit(1);
                }
                int delId = Integer.parseInt(opts.get("id"));
                if (client.deleteMessage(delId)) {
                    System.out.println("Письмо " + delId + " удалено.");
                } else {
                    System.out.println("Письмо не найдено.");
                }
                break;
            case "search":
                if (!opts.containsKey("query")) {
                    System.err.println("Требуется --query");
                    System.exit(1);
                }
                List<Message> results = client.search(opts.get("query"));
                if (results.isEmpty()) {
                    System.out.println("Ничего не найдено.");
                } else {
                    for (Message m : results) {
                        String status = m.read ? "🔵" : "⚪";
                        System.out.printf("%s %s | %s | %s | %s\n",
                                status,
                                colorize(String.valueOf(m.id), YELLOW),
                                colorize(m.from, GREEN),
                                colorize(m.subject, CYAN),
                                m.date.substring(0,10));
                    }
                }
                break;
            case "export":
                String filename = opts.getOrDefault("file", "export.mbox");
                if (client.export(filename)) {
                    System.out.println("Экспорт завершён в " + filename);
                } else {
                    System.out.println("Ошибка экспорта (нет активного аккаунта).");
                }
                break;
            case "accounts":
                List<Account> accs = client.listAccounts();
                String cur = client.getCurrentEmail();
                if (accs.isEmpty()) {
                    System.out.println("Нет аккаунтов.");
                } else {
                    for (Account a : accs) {
                        String marker = a.email.equals(cur) ? " *" : "";
                        System.out.println(a.email + " (" + a.name + ")" + marker);
                    }
                }
                break;
            default:
                System.out.println("Неизвестная команда.");
        }
    }
}
