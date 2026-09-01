# mail.py
# Unified Inbox на Python

import sys
import json
import os
import datetime
import argparse
from typing import List, Dict, Any, Optional

# ANSI-цвета
RESET = "\033[0m"
GREEN = "\033[92m"
YELLOW = "\033[93m"
CYAN = "\033[96m"
RED = "\033[91m"
BOLD = "\033[1m"

def colorize(text: str, color: str) -> str:
    return f"{color}{text}{RESET}"

class MailClient:
    def __init__(self, data_file="mailbox.json"):
        self.data_file = data_file
        self.data = self._load()
        self.current_email = self.data.get("current")

    def _load(self) -> Dict:
        if os.path.exists(self.data_file):
            with open(self.data_file, 'r', encoding='utf-8') as f:
                return json.load(f)
        return {"accounts": [], "current": None}

    def _save(self):
        with open(self.data_file, 'w', encoding='utf-8') as f:
            json.dump(self.data, f, indent=2, ensure_ascii=False)

    def _get_account(self, email: str) -> Optional[Dict]:
        for acc in self.data["accounts"]:
            if acc["email"] == email:
                return acc
        return None

    def _get_current_account(self) -> Optional[Dict]:
        if not self.current_email:
            return None
        return self._get_account(self.current_email)

    def add_account(self, name: str, email: str) -> bool:
        if self._get_account(email):
            return False
        acc = {
            "name": name,
            "email": email,
            "messages": [],
            "next_id": 1,
            "sent": [],
            "trash": []
        }
        self.data["accounts"].append(acc)
        if not self.current_email:
            self.data["current"] = email
            self.current_email = email
        self._save()
        return True

    def switch_account(self, email: str) -> bool:
        if self._get_account(email):
            self.data["current"] = email
            self.current_email = email
            self._save()
            return True
        return False

    def list_messages(self) -> List[Dict]:
        acc = self._get_current_account()
        if not acc:
            return []
        # Возвращаем все письма из inbox, sent, trash (для простоты показываем все)
        all_msgs = acc.get("messages", []) + acc.get("sent", []) + acc.get("trash", [])
        # Сортируем по дате (новые сверху)
        return sorted(all_msgs, key=lambda m: m.get("date", ""), reverse=True)

    def get_message(self, msg_id: int) -> Optional[Dict]:
        acc = self._get_current_account()
        if not acc:
            return None
        for folder in ["messages", "sent", "trash"]:
            for msg in acc.get(folder, []):
                if msg["id"] == msg_id:
                    msg["read"] = True
                    self._save()
                    return msg
        return None

    def send_message(self, to: str, subject: str, body: str) -> bool:
        acc = self._get_current_account()
        if not acc:
            return False
        msg = {
            "id": acc["next_id"],
            "from": acc["email"],
            "to": to,
            "subject": subject,
            "body": body,
            "date": datetime.datetime.now().isoformat(),
            "read": True
        }
        acc["next_id"] += 1
        acc.setdefault("sent", []).append(msg)
        self._save()
        return True

    def delete_message(self, msg_id: int) -> bool:
        acc = self._get_current_account()
        if not acc:
            return False
        for folder in ["messages", "sent"]:
            for i, msg in enumerate(acc.get(folder, [])):
                if msg["id"] == msg_id:
                    acc.setdefault("trash", []).append(msg)
                    del acc[folder][i]
                    self._save()
                    return True
        return False

    def search(self, query: str) -> List[Dict]:
        acc = self._get_current_account()
        if not acc:
            return []
        results = []
        for folder in ["messages", "sent", "trash"]:
            for msg in acc.get(folder, []):
                if (query.lower() in msg.get("subject", "").lower() or
                    query.lower() in msg.get("body", "").lower() or
                    query.lower() in msg.get("from", "").lower() or
                    query.lower() in msg.get("to", "").lower()):
                    results.append(msg)
        return sorted(results, key=lambda m: m.get("date", ""), reverse=True)

    def export(self, filename: str) -> bool:
        acc = self._get_current_account()
        if not acc:
            return False
        all_msgs = acc.get("messages", []) + acc.get("sent", []) + acc.get("trash", [])
        # Экспорт в формате MBOX (упрощённо)
        with open(filename, 'w', encoding='utf-8') as f:
            for msg in all_msgs:
                f.write(f"From {msg.get('from', '')} {msg.get('date', '')}\n")
                f.write(f"To: {msg.get('to', '')}\n")
                f.write(f"Subject: {msg.get('subject', '')}\n")
                f.write(f"\n{msg.get('body', '')}\n\n")
        return True

    def list_accounts(self):
        return self.data["accounts"]

    def get_current_email(self):
        return self.current_email

def main():
    parser = argparse.ArgumentParser(description="Unified Inbox - Email Client")
    parser.add_argument("command", choices=["add-account", "switch", "list", "read", "send", "delete", "search", "export", "accounts", "help"],
                        help="Команда")
    parser.add_argument("--name", help="Имя аккаунта")
    parser.add_argument("--email", help="Email адрес")
    parser.add_argument("--id", type=int, help="ID письма")
    parser.add_argument("--to", help="Получатель")
    parser.add_argument("--subject", help="Тема")
    parser.add_argument("--body", help="Текст")
    parser.add_argument("--query", help="Поисковый запрос")
    parser.add_argument("--file", help="Имя файла для экспорта")
    parser.add_argument("--data", default="mailbox.json", help="Файл данных")
    args = parser.parse_args()

    client = MailClient(args.data)

    if args.command == "help":
        print(__doc__)
        sys.exit(0)

    elif args.command == "add-account":
        if not args.name or not args.email:
            print("Требуются --name и --email")
            sys.exit(1)
        if client.add_account(args.name, args.email):
            print(f"Аккаунт {args.email} добавлен.")
        else:
            print(f"Аккаунт {args.email} уже существует.")

    elif args.command == "switch":
        if not args.email:
            print("Требуется --email")
            sys.exit(1)
        if client.switch_account(args.email):
            print(f"Переключено на {args.email}")
        else:
            print(f"Аккаунт {args.email} не найден.")

    elif args.command == "list":
        msgs = client.list_messages()
        if not msgs:
            print("Нет писем.")
        else:
            current = client.get_current_email()
            print(colorize(f"Почта: {current}", BOLD + CYAN))
            for msg in msgs:
                status = "🔵" if msg.get("read") else "⚪"
                print(f"{status} {colorize(str(msg['id']), YELLOW)} | {colorize(msg.get('from', ''), GREEN)} | {colorize(msg.get('subject', ''), CYAN)} | {msg.get('date', '')[:10]}")

    elif args.command == "read":
        if not args.id:
            print("Требуется --id")
            sys.exit(1)
        msg = client.get_message(args.id)
        if not msg:
            print("Письмо не найдено.")
        else:
            print(colorize(f"От: {msg.get('from', '')}", GREEN))
            print(colorize(f"Кому: {msg.get('to', '')}", GREEN))
            print(colorize(f"Тема: {msg.get('subject', '')}", CYAN))
            print(colorize(f"Дата: {msg.get('date', '')}", YELLOW))
            print("\n" + msg.get("body", ""))

    elif args.command == "send":
        if not args.to or not args.subject or not args.body:
            print("Требуются --to, --subject, --body")
            sys.exit(1)
        if client.send_message(args.to, args.subject, args.body):
            print("Письмо отправлено.")
        else:
            print("Ошибка отправки (нет активного аккаунта).")

    elif args.command == "delete":
        if not args.id:
            print("Требуется --id")
            sys.exit(1)
        if client.delete_message(args.id):
            print(f"Письмо {args.id} удалено.")
        else:
            print("Письмо не найдено.")

    elif args.command == "search":
        if not args.query:
            print("Требуется --query")
            sys.exit(1)
        results = client.search(args.query)
        if not results:
            print("Ничего не найдено.")
        else:
            for msg in results:
                status = "🔵" if msg.get("read") else "⚪"
                print(f"{status} {colorize(str(msg['id']), YELLOW)} | {colorize(msg.get('from', ''), GREEN)} | {colorize(msg.get('subject', ''), CYAN)} | {msg.get('date', '')[:10]}")

    elif args.command == "export":
        filename = args.file or "export.mbox"
        if client.export(filename):
            print(f"Экспорт завершён в {filename}")
        else:
            print("Ошибка экспорта (нет активного аккаунта).")

    elif args.command == "accounts":
        accs = client.list_accounts()
        current = client.get_current_email()
        if not accs:
            print("Нет аккаунтов.")
        else:
            for acc in accs:
                marker = " *" if acc["email"] == current else ""
                print(f"{acc['email']} ({acc['name']}){marker}")

if __name__ == "__main__":
    main()
