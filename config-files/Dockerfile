# Використовуємо офіційний образ Python 3.12 slim як базовий
FROM python:3.12-slim

# Оновлюємо пакетний менеджер та встановлюємо git, потім очищаємо кеш apt
RUN apt-get update && apt-get install -y git rm -rf /var/lib/apt/lists/*

# Клонуємо репозиторій Cowrie з GitHub до директорії /cowrie
RUN git clone https://github.com/cowrie/cowrie.git /cowrie

# Встановлюємо робочу директорію /cowrie для наступних команд
WORKDIR /cowrie

# Встановлюємо залежності Python з requirements.txt без кешування
RUN pip install --no-cache-dir -r requirements.txt

# Копіюємо локальний конфігураційний файл cowrie.cfg до контейнера
COPY cowrie.cfg /cowrie/etc/cowrie.cfg

# Відкриваємо порт 2222 для SSH-підключень
EXPOSE 2222

# Запускаємо Cowrie при старті контейнера
CMD ["bin/cowrie", "start"]