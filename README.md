# Pull request review assignment service
Микросервис, который автоматически назначает ревьюеров на Pull Request’ы, а также позволяет управлять командами и участниками. 

# Технологии
- Go
- Docker & Docker compose
- PostgreSQL + миграции
- Makefile


# Установка и запуск
**Установка:**
```bash
git clone https://github.com/vedsatt/pr-review-assignment-service
cd pr-review-assignment-service
```
**Запуск:**
```bash
make build
```
или
```bash
docker-compose --env-file config/.env up -d --build
```

# Вопросы и решения 
Вопросы, которые возникли у меня во время написания сервиса и то, как я их решил.
## Ошибки
Сервис возвращал слишком мало ошибок, поэтому я добавил несколько дополнительных:
```go
// internal/models/errors.go:
UserExistsErr  string = "USER_EXISTS"
InvalidJSONErr string = "INVALID_JSON"
InternalErr    string = "NTERNAL_ERROR"
```
Также я добавил простую обработку этих ошибок на слое сервиса

## Переназначение ревьюеров
Я не знал, как лучше настроить автоматическое (когда пользователь меняет статус ревьюера на false) и ручное (через /pullRequest/reassign) переназначение ревьюеров. Я решил разделить логику на две части:

Если пользователь меняет статус ревьюера на false, но кандидатов на его место для ревью нет - мы удаляем его из ревьюеров без замены.

Eсли же мы хотим напрямую поменять ревьюера, но кандидатов нет - сервер не даст нам этого сделать.

# Доп задания:
## Эндпоинты статистики:
### user statistics
Статистика:
- кол-во всех пользователей
- кол-во активных пользователей
- кол-во неактивных пользователей
- пользователей в каждой команде

Запрос:
```
GET /statistics/users
```
Ответ:
```json
{
    "total_users": 7,
    "active_users": 6,
    "inactive_users": 1,
    "users_by_team": [
        {
            "team_name": "payments",
            "users_count": 4
        },
        {
            "team_name": "backend",
            "users_count": 3
        }
    ]
}
```
### pull requests statistics:
Статистика:
- кол-во пров
- кол-во открытых пл-ов
- кол-во закрытых пл-ов

Запрос:
```
GET /statistics/pullRequests
```
Ответ:
```json
{
    "total_prs": 3,
    "open_prs": 2,
    "merged_prs": 1
}

```
### reviewers statistics:
Статистика:
- топ 10 (максимум) ревьюера
- пользователи без ревью

Запрос:
```
GET /statistics/reviewers
```
Ответ:
```json
{
    "top_reviewers": [
        {
            "user_id": "u2",
            "username": "Bob",
            "review_count": 1
        },
        {
            "user_id": "u1",
            "username": "Alice",
            "review_count": 1
        },
        {
            "user_id": "u13",
            "username": "Liza",
            "review_count": 1
        },
        {
            "user_id": "u3",
            "username": "Greg",
            "review_count": 1
        },
        {
            "user_id": "u4",
            "username": "Anton",
            "review_count": 1
        }
    ],
    "users_without_reviews": [
        "u11", 
        "u12"
    ]
}
```

