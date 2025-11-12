# Backend для хакатона осень 2025

## Запуск

### Запуск для разработчика (конфиг по дефолту [config.template.yml](config/config.template.yml))
```bash
# Запускаем зависимые сервисы
docker compose -f docker-compose.dev.yml up -d

# Генерируем .pem ключи
task keygen

# Запуск приложения
task run 
```

### Запуск для фронтендера (конфиг по дефолту [config.docker.yml](config/config.docker.yml))
```bash
# Клонируем репозиторий 
git clone https://github.com/DenisBochko/hackathon-back.git

# Переходим в директорию проекта
cd hackathon-back

# Запускаем в Docker и всё
docker compose up -d --build 

# Можно потыкать конфиги, но это не рекомендуется 
```
