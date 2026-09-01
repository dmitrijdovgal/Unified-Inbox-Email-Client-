# mail.rb
# Unified Inbox на Ruby

require 'json'
require 'time'

# ANSI-цвета
RESET = "\033[0m"
GREEN = "\033[92m"
YELLOW = "\033[93m"
CYAN = "\033[96m"
RED = "\033[91m"
BOLD = "\033[1m"

def colorize(text, color)
  "#{color}#{text}#{RESET}"
end

class MailClient
  def initialize(data_file = 'mailbox.json')
    @data_file = data_file
    load_data
  end

  def load_data
    if File.exist?(@data_file)
      @data = JSON.parse(File.read(@data_file), symbolize_names: true)
    else
      @data = { accounts: [], current: nil }
    end
    @current_email = @data[:current]
  end

  def save_data
    File.write(@data_file, JSON.pretty_generate(@data))
  end

  def get_account(email)
    @data[:accounts].find { |a| a[:email] == email }
  end

  def get_current_account
    return nil unless @current_email
    get_account(@current_email)
  end

  def add_account(name, email)
    return false if get_account(email)
    acc = {
      name: name,
      email: email,
      messages: [],
      sent: [],
      trash: [],
      next_id: 1
    }
    @data[:accounts] << acc
    @data[:current] ||= email
    @current_email = @data[:current]
    save_data
    true
  end

  def switch_account(email)
    return false unless get_account(email)
    @data[:current] = email
    @current_email = email
    save_data
    true
  end

  def list_messages
    acc = get_current_account
    return [] unless acc
    all = acc[:messages] + acc[:sent] + acc[:trash]
    all.sort_by! { |m| m[:date] }.reverse!
  end

  def get_message(id)
    acc = get_current_account
    return nil unless acc
    [:messages, :sent, :trash].each do |folder|
      acc[folder].each do |msg|
        if msg[:id] == id
          msg[:read] = true
          save_data
          return msg
        end
      end
    end
    nil
  end

  def send_message(to, subject, body)
    acc = get_current_account
    return false unless acc
    msg = {
      id: acc[:next_id],
      from: acc[:email],
      to: to,
      subject: subject,
      body: body,
      date: Time.now.iso8601,
      read: true
    }
    acc[:next_id] += 1
    acc[:sent] << msg
    save_data
    true
  end

  def delete_message(id)
    acc = get_current_account
    return false unless acc
    [:messages, :sent].each do |folder|
      list = acc[folder]
      idx = list.index { |m| m[:id] == id }
      if idx
        acc[:trash] << list.delete_at(idx)
        save_data
        return true
      end
    end
    false
  end

  def search(query)
    acc = get_current_account
    return [] unless acc
    q = query.downcase
    results = []
    [:messages, :sent, :trash].each do |folder|
      acc[folder].each do |msg|
        if msg[:subject].downcase.include?(q) ||
           msg[:body].downcase.include?(q) ||
           msg[:from].downcase.include?(q) ||
           msg[:to].downcase.include?(q)
          results << msg
        end
      end
    end
    results.sort_by! { |m| m[:date] }.reverse!
  end

  def export(filename)
    acc = get_current_account
    return false unless acc
    all = acc[:messages] + acc[:sent] + acc[:trash]
    content = all.map do |msg|
      "From #{msg[:from]} #{msg[:date]}\nTo: #{msg[:to]}\nSubject: #{msg[:subject]}\n\n#{msg[:body]}\n\n"
    end.join
    File.write(filename, content)
    true
  end

  def list_accounts
    @data[:accounts]
  end

  def get_current_email
    @current_email
  end
end

if ARGV.empty? || ARGV[0] == 'help'
  puts <<~HELP
    Использование: ruby mail.rb <команда> [опции]
      add-account   --name <name> --email <email>
      switch        --email <email>
      list
      read          --id <id>
      send          --to <to> --subject <subject> --body <body>
      delete        --id <id>
      search        --query <query>
      export        [--file <file>]
      accounts
      help
  HELP
  exit
end

cmd = ARGV[0]
opts = {}
i = 1
while i < ARGV.length
  if ARGV[i].start_with?('--')
    key = ARGV[i][2..-1]
    if i + 1 < ARGV.length && !ARGV[i+1].start_with?('--')
      opts[key] = ARGV[i+1]
      i += 2
    else
      opts[key] = ''
      i += 1
    end
  else
    i += 1
  end
end

data_file = opts['data'] || 'mailbox.json'
client = MailClient.new(data_file)

case cmd
when 'add-account'
  unless opts['name'] && opts['email']
    puts "Требуются --name и --email"
    exit 1
  end
  if client.add_account(opts['name'], opts['email'])
    puts "Аккаунт #{opts['email']} добавлен."
  else
    puts "Аккаунт #{opts['email']} уже существует."
  end
when 'switch'
  unless opts['email']
    puts "Требуется --email"
    exit 1
  end
  if client.switch_account(opts['email'])
    puts "Переключено на #{opts['email']}"
  else
    puts "Аккаунт #{opts['email']} не найден."
  end
when 'list'
  msgs = client.list_messages
  if msgs.empty?
    puts "Нет писем."
  else
    current = client.get_current_email
    puts colorize("Почта: #{current}", BOLD + CYAN)
    msgs.each do |m|
      status = m[:read] ? '🔵' : '⚪'
      puts "#{status} #{colorize(m[:id].to_s, YELLOW)} | #{colorize(m[:from], GREEN)} | #{colorize(m[:subject], CYAN)} | #{m[:date][0,10]}"
    end
  end
when 'read'
  unless opts['id']
    puts "Требуется --id"
    exit 1
  end
  msg = client.get_message(opts['id'].to_i)
  if msg.nil?
    puts "Письмо не найдено."
  else
    puts colorize("От: #{msg[:from]}", GREEN)
    puts colorize("Кому: #{msg[:to]}", GREEN)
    puts colorize("Тема: #{msg[:subject]}", CYAN)
    puts colorize("Дата: #{msg[:date]}", YELLOW)
    puts "\n#{msg[:body]}"
  end
when 'send'
  unless opts['to'] && opts['subject'] && opts['body']
    puts "Требуются --to, --subject, --body"
    exit 1
  end
  if client.send_message(opts['to'], opts['subject'], opts['body'])
    puts "Письмо отправлено."
  else
    puts "Ошибка отправки (нет активного аккаунта)."
  end
when 'delete'
  unless opts['id']
    puts "Требуется --id"
    exit 1
  end
  if client.delete_message(opts['id'].to_i)
    puts "Письмо #{opts['id']} удалено."
  else
    puts "Письмо не найдено."
  end
when 'search'
  unless opts['query']
    puts "Требуется --query"
    exit 1
  end
  results = client.search(opts['query'])
  if results.empty?
    puts "Ничего не найдено."
  else
    results.each do |m|
      status = m[:read] ? '🔵' : '⚪'
      puts "#{status} #{colorize(m[:id].to_s, YELLOW)} | #{colorize(m[:from], GREEN)} | #{colorize(m[:subject], CYAN)} | #{m[:date][0,10]}"
    end
  end
when 'export'
  filename = opts['file'] || 'export.mbox'
  if client.export(filename)
    puts "Экспорт завершён в #{filename}"
  else
    puts "Ошибка экспорта (нет активного аккаунта)."
  end
when 'accounts'
  accs = client.list_accounts
  current = client.get_current_email
  if accs.empty?
    puts "Нет аккаунтов."
  else
    accs.each do |a|
      marker = a[:email] == current ? ' *' : ''
      puts "#{a[:email]} (#{a[:name]})#{marker}"
    end
  end
else
  puts "Неизвестная команда."
end
