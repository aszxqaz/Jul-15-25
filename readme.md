# API загрузки и архивирования файлов

## Старт

```bash
cd ./cmd/api
go run main.go
bin/main
```

## Эндпоинты

1. Добавить задачу: `POST http://localhost:8080/api/archives`.

2. Узнать статус задачи: `GET http://localhost:8080/api/archives/{id задачи}`.

3. Добавить в задачу ссылки на файлы: `POST http://localhost:8080/api/archives/{id задачи}/files`.
   Запрос должен включать тело в формате JSON, содержащее объект с полем `urls`, имеющим тип массива строк:

```
{
  "urls": [
    "www.site1.com/file1.jpg",
    "www.site2.com/file2.jpg"
  ]
}
```
