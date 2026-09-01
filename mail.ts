// mail.ts
// Unified Inbox на TypeScript

import * as fs from 'fs';

// ANSI-цвета
const RESET = '\x1b[0m';
const GREEN = '\x1b[92m';
const YELLOW = '\x1b[93m';
const CYAN = '\x1b[96m';
const RED = '\x1b[91m';
const BOLD = '\x1b[1m';

function colorize(text: string, color: string): string {
    return `${color}${text}${RESET}`;
}

interface Message {
    id: number;
    from: string;
    to: string;
    subject: string;
    body: string;
    date: string;
    read: boolean;
}

interface Account {
    name: string;
    email: string;
    messages: Message[];
    sent: Message[];
    trash: Message[];
    next_id: number;
}

interface Data {
    accounts: Account[];
    current: string | null;
}

class MailClient {
    private data: Data;
    private currentEmail: string | null;
    private dataFile: string;

    constructor(dataFile: string = 'mailbox.json') {
        this.dataFile = dataFile;
        this.data = this.load();
        this.currentEmail = this.data.current;
    }

    private load(): Data {
        try {
            if (fs.existsSync(this.dataFile)) {
                const content = fs.readFileSync(this.dataFile, 'utf-8');
                return JSON.parse(content);
            }
        } catch (e) {}
        return { accounts: [], current: null };
    }

    private save(): void {
        fs.writeFileSync(this.dataFile, JSON.stringify(this.data, null, 2), 'utf-8');
    }

    private getAccount(email: string): Account | undefined {
        return this.data.accounts.find(a => a.email === email);
    }

    private getCurrentAccount(): Account | undefined {
        if (!this.currentEmail) return undefined;
        return this.getAccount(this.currentEmail);
    }

    addAccount(name: string, email: string): boolean {
        if (this.getAccount(email)) return false;
        const acc: Account = {
            name,
            email,
            messages: [],
            sent: [],
            trash: [],
            next_id: 1
        };
        this.data.accounts.push(acc);
        if (!this.currentEmail) {
            this.data.current = email;
            this.currentEmail = email;
        }
        this.save();
        return true;
    }

    switchAccount(email: string): boolean {
        if (this.getAccount(email)) {
            this.data.current = email;
            this.currentEmail = email;
            this.save();
            return true;
        }
        return false;
    }

    listMessages(): Message[] {
        const acc = this.getCurrentAccount();
        if (!acc) return [];
        const all = [...acc.messages, ...acc.sent, ...acc.trash];
        return all.sort((a, b) => (a.date < b.date ? 1 : -1));
    }

    getMessage(id: number): Message | null {
        const acc = this.getCurrentAccount();
        if (!acc) return null;
        for (const folder of ['messages', 'sent', 'trash'] as const) {
            const list = acc[folder] || [];
            for (const msg of list) {
                if (msg.id === id) {
                    msg.read = true;
                    this.save();
                    return msg;
                }
            }
        }
        return null;
    }

    sendMessage(to: string, subject: string, body: string): boolean {
        const acc = this.getCurrentAccount();
        if (!acc) return false;
        const msg: Message = {
            id: acc.next_id++,
            from: acc.email,
            to,
            subject,
            body,
            date: new Date().toISOString(),
            read: true
        };
        acc.sent.push(msg);
        this.save();
        return true;
    }

    deleteMessage(id: number): boolean {
        const acc = this.getCurrentAccount();
        if (!acc) return false;
        for (const folder of ['messages', 'sent'] as const) {
            const list = acc[folder] || [];
            for (let i = 0; i < list.length; i++) {
                if (list[i].id === id) {
                    acc.trash.push(list[i]);
                    list.splice(i, 1);
                    this.save();
                    return true;
                }
            }
        }
        return false;
    }

    search(query: string): Message[] {
        const acc = this.getCurrentAccount();
        if (!acc) return [];
        const results: Message[] = [];
        const q = query.toLowerCase();
        for (const folder of ['messages', 'sent', 'trash'] as const) {
            for (const msg of acc[folder] || []) {
                if (msg.subject.toLowerCase().includes(q) ||
                    msg.body.toLowerCase().includes(q) ||
                    msg.from.toLowerCase().includes(q) ||
                    msg.to.toLowerCase().includes(q)) {
                    results.push(msg);
                }
            }
        }
        return results.sort((a, b) => (a.date < b.date ? 1 : -1));
    }

    export(filename: string): boolean {
        const acc = this.getCurrentAccount();
        if (!acc) return false;
        const all = [...acc.messages, ...acc.sent, ...acc.trash];
        const content = all.map(msg => {
            return `From ${msg.from} ${msg.date}\nTo: ${msg.to}\nSubject: ${msg.subject}\n\n${msg.body}\n\n`;
        }).join('');
        fs.writeFileSync(filename, content, 'utf-8');
        return true;
    }

    listAccounts(): Account[] {
        return this.data.accounts;
    }

    getCurrentEmail(): string | null {
        return this.currentEmail;
    }
}

function main(): void {
    const args = process.argv.slice(2);
    if (args.length === 0 || args[0] === 'help') {
        console.log(`Использование: ts-node mail.ts <команда> [опции]
  add-account   --name <name> --email <email>
  switch        --email <email>
  list
  read          --id <id>
  send          --to <to> --subject <subject> --body <body>
  delete        --id <id>
  search        --query <query>
  export        [--file <file>]
  accounts
  help`);
        process.exit(0);
    }
    const command = args[0];
    const options: Record<string, string> = {};
    for (let i = 1; i < args.length; i++) {
        const arg = args[i];
        if (arg.startsWith('--')) {
            const key = arg.slice(2);
            const value = args[++i];
            options[key] = value;
        }
    }
    const dataFile = options.data || 'mailbox.json';
    const client = new MailClient(dataFile);

    switch (command) {
        case 'add-account': {
            if (!options.name || !options.email) {
                console.error('Требуются --name и --email');
                process.exit(1);
            }
            if (client.addAccount(options.name, options.email)) {
                console.log(`Аккаунт ${options.email} добавлен.`);
            } else {
                console.log(`Аккаунт ${options.email} уже существует.`);
            }
            break;
        }
        case 'switch': {
            if (!options.email) {
                console.error('Требуется --email');
                process.exit(1);
            }
            if (client.switchAccount(options.email)) {
                console.log(`Переключено на ${options.email}`);
            } else {
                console.log(`Аккаунт ${options.email} не найден.`);
            }
            break;
        }
        case 'list': {
            const msgs = client.listMessages();
            if (!msgs.length) {
                console.log('Нет писем.');
            } else {
                const current = client.getCurrentEmail();
                console.log(colorize(`Почта: ${current}`, BOLD + CYAN));
                for (const msg of msgs) {
                    const status = msg.read ? '🔵' : '⚪';
                    console.log(`${status} ${colorize(String(msg.id), YELLOW)} | ${colorize(msg.from, GREEN)} | ${colorize(msg.subject, CYAN)} | ${msg.date.slice(0,10)}`);
                }
            }
            break;
        }
        case 'read': {
            if (!options.id) {
                console.error('Требуется --id');
                process.exit(1);
            }
            const msg = client.getMessage(parseInt(options.id));
            if (!msg) {
                console.log('Письмо не найдено.');
            } else {
                console.log(colorize(`От: ${msg.from}`, GREEN));
                console.log(colorize(`Кому: ${msg.to}`, GREEN));
                console.log(colorize(`Тема: ${msg.subject}`, CYAN));
                console.log(colorize(`Дата: ${msg.date}`, YELLOW));
                console.log('\n' + msg.body);
            }
            break;
        }
        case 'send': {
            if (!options.to || !options.subject || !options.body) {
                console.error('Требуются --to, --subject, --body');
                process.exit(1);
            }
            if (client.sendMessage(options.to, options.subject, options.body)) {
                console.log('Письмо отправлено.');
            } else {
                console.log('Ошибка отправки (нет активного аккаунта).');
            }
            break;
        }
        case 'delete': {
            if (!options.id) {
                console.error('Требуется --id');
                process.exit(1);
            }
            if (client.deleteMessage(parseInt(options.id))) {
                console.log(`Письмо ${options.id} удалено.`);
            } else {
                console.log('Письмо не найдено.');
            }
            break;
        }
        case 'search': {
            if (!options.query) {
                console.error('Требуется --query');
                process.exit(1);
            }
            const results = client.search(options.query);
            if (!results.length) {
                console.log('Ничего не найдено.');
            } else {
                for (const msg of results) {
                    const status = msg.read ? '🔵' : '⚪';
                    console.log(`${status} ${colorize(String(msg.id), YELLOW)} | ${colorize(msg.from, GREEN)} | ${colorize(msg.subject, CYAN)} | ${msg.date.slice(0,10)}`);
                }
            }
            break;
        }
        case 'export': {
            const filename = options.file || 'export.mbox';
            if (client.export(filename)) {
                console.log(`Экспорт завершён в ${filename}`);
            } else {
                console.log('Ошибка экспорта (нет активного аккаунта).');
            }
            break;
        }
        case 'accounts': {
            const accs = client.listAccounts();
            const current = client.getCurrentEmail();
            if (!accs.length) {
                console.log('Нет аккаунтов.');
            } else {
                for (const acc of accs) {
                    const marker = acc.email === current ? ' *' : '';
                    console.log(`${acc.email} (${acc.name})${marker}`);
                }
            }
            break;
        }
        default:
            console.log('Неизвестная команда.');
    }
}

main();
