<?php
// mail.php
// Unified Inbox на PHP

if (php_sapi_name() !== 'cli') {
    die("Это консольное приложение.\n");
}

// ANSI-цвета
define('RESET', "\033[0m");
define('GREEN', "\033[92m");
define('YELLOW', "\033[93m");
define('CYAN', "\033[96m");
define('RED', "\033[91m");
define('BOLD', "\033[1m");

function colorize($text, $color) {
    return $color . $text . RESET;
}

class MailClient {
    private $dataFile;
    private $data;
    private $currentEmail;

    public function __construct($dataFile = 'mailbox.json') {
        $this->dataFile = $dataFile;
        $this->load();
    }

    private function load() {
        if (!file_exists($this->dataFile)) {
            $this->data = ['accounts' => [], 'current' => null];
            return;
        }
        $json = file_get_contents($this->dataFile);
        $this->data = json_decode($json, true);
        if (!$this->data) {
            $this->data = ['accounts' => [], 'current' => null];
        }
        $this->currentEmail = $this->data['current'] ?? null;
    }

    private function save() {
        file_put_contents($this->dataFile, json_encode($this->data, JSON_PRETTY_PRINT | JSON_UNESCAPED_UNICODE));
    }

    private function getAccount($email) {
        foreach ($this->data['accounts'] as &$acc) {
            if ($acc['email'] == $email) return $acc;
        }
        return null;
    }

    private function getCurrentAccount() {
        if (!$this->currentEmail) return null;
        return $this->getAccount($this->currentEmail);
    }

    public function addAccount($name, $email) {
        if ($this->getAccount($email)) return false;
        $acc = [
            'name' => $name,
            'email' => $email,
            'messages' => [],
            'sent' => [],
            'trash' => [],
            'next_id' => 1
        ];
        $this->data['accounts'][] = $acc;
        if (!$this->currentEmail) {
            $this->data['current'] = $email;
            $this->currentEmail = $email;
        }
        $this->save();
        return true;
    }

    public function switchAccount($email) {
        if (!$this->getAccount($email)) return false;
        $this->data['current'] = $email;
        $this->currentEmail = $email;
        $this->save();
        return true;
    }

    public function listMessages() {
        $acc = $this->getCurrentAccount();
        if (!$acc) return [];
        $all = array_merge($acc['messages'], $acc['sent'], $acc['trash']);
        usort($all, function($a, $b) {
            return strcmp($b['date'], $a['date']);
        });
        return $all;
    }

    public function getMessage($id) {
        $acc = $this->getCurrentAccount();
        if (!$acc) return null;
        foreach (['messages', 'sent', 'trash'] as $folder) {
            foreach ($acc[$folder] as &$msg) {
                if ($msg['id'] == $id) {
                    $msg['read'] = true;
                    $this->save();
                    return $msg;
                }
            }
        }
        return null;
    }

    public function sendMessage($to, $subject, $body) {
        $acc = $this->getCurrentAccount();
        if (!$acc) return false;
        $msg = [
            'id' => $acc['next_id']++,
            'from' => $acc['email'],
            'to' => $to,
            'subject' => $subject,
            'body' => $body,
            'date' => date('c'),
            'read' => true
        ];
        $acc['sent'][] = $msg;
        $this->save();
        return true;
    }

    public function deleteMessage($id) {
        $acc = $this->getCurrentAccount();
        if (!$acc) return false;
        foreach (['messages', 'sent'] as $folder) {
            foreach ($acc[$folder] as $i => $msg) {
                if ($msg['id'] == $id) {
                    $acc['trash'][] = $msg;
                    unset($acc[$folder][$i]);
                    $acc[$folder] = array_values($acc[$folder]);
                    $this->save();
                    return true;
                }
            }
        }
        return false;
    }

    public function search($query) {
        $acc = $this->getCurrentAccount();
        if (!$acc) return [];
        $q = strtolower($query);
        $results = [];
        foreach (['messages', 'sent', 'trash'] as $folder) {
            foreach ($acc[$folder] as $msg) {
                if (strpos(strtolower($msg['subject']), $q) !== false ||
                    strpos(strtolower($msg['body']), $q) !== false ||
                    strpos(strtolower($msg['from']), $q) !== false ||
                    strpos(strtolower($msg['to']), $q) !== false) {
                    $results[] = $msg;
                }
            }
        }
        usort($results, function($a, $b) {
            return strcmp($b['date'], $a['date']);
        });
        return $results;
    }

    public function export($filename) {
        $acc = $this->getCurrentAccount();
        if (!$acc) return false;
        $all = array_merge($acc['messages'], $acc['sent'], $acc['trash']);
        $content = '';
        foreach ($all as $msg) {
            $content .= "From {$msg['from']} {$msg['date']}\n";
            $content .= "To: {$msg['to']}\n";
            $content .= "Subject: {$msg['subject']}\n";
            $content .= "\n{$msg['body']}\n\n";
        }
        file_put_contents($filename, $content);
        return true;
    }

    public function listAccounts() {
        return $this->data['accounts'];
    }

    public function getCurrentEmail() {
        return $this->currentEmail;
    }
}

$args = array_slice($argv, 1);
if (empty($args) || $args[0] == 'help') {
    echo "Использование: php mail.php <команда> [опции]\n";
    echo "  add-account   --name <name> --email <email>\n";
    echo "  switch        --email <email>\n";
    echo "  list\n";
    echo "  read          --id <id>\n";
    echo "  send          --to <to> --subject <subject> --body <body>\n";
    echo "  delete        --id <id>\n";
    echo "  search        --query <query>\n";
    echo "  export        [--file <file>]\n";
    echo "  accounts\n";
    echo "  help\n";
    exit(0);
}
$cmd = $args[0];
$opts = [];
for ($i = 1; $i < count($args); $i++) {
    if (strpos($args[$i], '--') === 0) {
        $key = substr($args[$i], 2);
        if (isset($args[$i+1]) && strpos($args[$i+1], '--') !== 0) {
            $opts[$key] = $args[++$i];
        } else {
            $opts[$key] = '';
        }
    }
}
$dataFile = $opts['data'] ?? 'mailbox.json';
$client = new MailClient($dataFile);

switch ($cmd) {
    case 'add-account':
        if (empty($opts['name']) || empty($opts['email'])) {
            echo "Требуются --name и --email\n";
            exit(1);
        }
        if ($client->addAccount($opts['name'], $opts['email'])) {
            echo "Аккаунт {$opts['email']} добавлен.\n";
        } else {
            echo "Аккаунт {$opts['email']} уже существует.\n";
        }
        break;
    case 'switch':
        if (empty($opts['email'])) {
            echo "Требуется --email\n";
            exit(1);
        }
        if ($client->switchAccount($opts['email'])) {
            echo "Переключено на {$opts['email']}\n";
        } else {
            echo "Аккаунт {$opts['email']} не найден.\n";
        }
        break;
    case 'list':
        $msgs = $client->listMessages();
        if (empty($msgs)) {
            echo "Нет писем.\n";
        } else {
            $current = $client->getCurrentEmail();
            echo colorize("Почта: $current", BOLD . CYAN) . "\n";
            foreach ($msgs as $msg) {
                $status = $msg['read'] ? '🔵' : '⚪';
                echo "$status " . colorize($msg['id'], YELLOW) . " | " . colorize($msg['from'], GREEN) . " | " . colorize($msg['subject'], CYAN) . " | " . substr($msg['date'], 0, 10) . "\n";
            }
        }
        break;
    case 'read':
        if (empty($opts['id'])) {
            echo "Требуется --id\n";
            exit(1);
        }
        $msg = $client->getMessage((int)$opts['id']);
        if (!$msg) {
            echo "Письмо не найдено.\n";
        } else {
            echo colorize("От: {$msg['from']}", GREEN) . "\n";
            echo colorize("Кому: {$msg['to']}", GREEN) . "\n";
            echo colorize("Тема: {$msg['subject']}", CYAN) . "\n";
            echo colorize("Дата: {$msg['date']}", YELLOW) . "\n";
            echo "\n{$msg['body']}\n";
        }
        break;
    case 'send':
        if (empty($opts['to']) || empty($opts['subject']) || empty($opts['body'])) {
            echo "Требуются --to, --subject, --body\n";
            exit(1);
        }
        if ($client->sendMessage($opts['to'], $opts['subject'], $opts['body'])) {
            echo "Письмо отправлено.\n";
        } else {
            echo "Ошибка отправки (нет активного аккаунта).\n";
        }
        break;
    case 'delete':
        if (empty($opts['id'])) {
            echo "Требуется --id\n";
            exit(1);
        }
        if ($client->deleteMessage((int)$opts['id'])) {
            echo "Письмо {$opts['id']} удалено.\n";
        } else {
            echo "Письмо не найдено.\n";
        }
        break;
    case 'search':
        if (empty($opts['query'])) {
            echo "Требуется --query\n";
            exit(1);
        }
        $results = $client->search($opts['query']);
        if (empty($results)) {
            echo "Ничего не найдено.\n";
        } else {
            foreach ($results as $msg) {
                $status = $msg['read'] ? '🔵' : '⚪';
                echo "$status " . colorize($msg['id'], YELLOW) . " | " . colorize($msg['from'], GREEN) . " | " . colorize($msg['subject'], CYAN) . " | " . substr($msg['date'], 0, 10) . "\n";
            }
        }
        break;
    case 'export':
        $filename = $opts['file'] ?? 'export.mbox';
        if ($client->export($filename)) {
            echo "Экспорт завершён в $filename\n";
        } else {
            echo "Ошибка экспорта (нет активного аккаунта).\n";
        }
        break;
    case 'accounts':
        $accs = $client->listAccounts();
        $cur = $client->getCurrentEmail();
        if (empty($accs)) {
            echo "Нет аккаунтов.\n";
        } else {
            foreach ($accs as $acc) {
                $marker = $acc['email'] == $cur ? ' *' : '';
                echo "{$acc['email']} ({$acc['name']})$marker\n";
            }
        }
        break;
    default:
        echo "Неизвестная команда.\n";
}
