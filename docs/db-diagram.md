# Схема базы данных

8 таблиц. Миграции лежат в `migrations/` и применяются Docker-ом при первом создании базы (`make infra-reset` пересоздаёт её с нуля). Refresh-токены в БД не хранятся, они в Redis.

```mermaid
erDiagram
    roles ||--o{ users : "роль аккаунта"
    users ||--o| employees : "аккаунт сотрудника"
    departments ||--o{ employees : "работает в"
    positions ||--o{ employees : "занимает"
    employees ||--o{ salaries : "история зарплат"
    employees ||--o{ time_off : "заявки"
    users ||--o{ time_off : "approved_by"
    employees ||--o{ employee_position_history : "перемещения"
    departments ||--o{ employee_position_history : "отдел"
    positions ||--o{ employee_position_history : "должность"

    roles {
        serial id PK
        varchar code UK "hr или employee"
        varchar name
        timestamptz created_at
    }
    users {
        serial id PK
        int role_id FK
        varchar email UK "в нижнем регистре"
        varchar password_hash "bcrypt"
        bool is_active "false у уволенных"
        timestamptz created_at
        timestamptz updated_at
    }
    departments {
        serial id PK
        varchar name UK
        timestamptz created_at
        timestamptz updated_at
    }
    positions {
        serial id PK
        varchar name UK
        numeric base_salary "ориентир оклада, >= 0"
        timestamptz created_at
        timestamptz updated_at
    }
    employees {
        serial id PK
        int user_id FK, UK
        int department_id FK
        int position_id FK
        varchar first_name
        varchar last_name
        varchar phone "до 20 символов, не UNIQUE"
        date hire_date
        date termination_date "NULL, пока работает"
        varchar status "active, on_leave, terminated"
        timestamptz created_at
        timestamptz updated_at
    }
    salaries {
        serial id PK
        int employee_id FK
        numeric amount "> 0"
        date effective_date "UNIQUE с employee_id"
        timestamptz created_at
    }
    time_off {
        serial id PK
        int employee_id FK
        int approved_by FK "NULL, id аккаунта HR"
        varchar leave_type "vacation или sick"
        date start_date
        date end_date "не раньше start_date"
        varchar status "pending, approved, rejected"
        timestamptz created_at
        timestamptz updated_at
    }
    employee_position_history {
        serial id PK
        int employee_id FK
        int department_id FK
        int position_id FK
        date start_date
        date end_date "NULL у текущей записи"
        timestamptz created_at
    }
```

## Правила, которые держит база

| Правило | Как |
|---|---|
| email уникален и хранится в нижнем регистре | `UNIQUE` и `CHECK (email = lower(email))` |
| один аккаунт это один сотрудник | `UNIQUE (user_id)` в `employees` |
| нельзя удалить отдел или должность, на которых есть сотрудники | `ON DELETE RESTRICT` (приложение отвечает 409) |
| уволенного не удаляем, а помечаем | `status = 'terminated'`, `termination_date`, `users.is_active = false` |
| дата увольнения не раньше даты приёма | `CHECK` в `employees` |
| у сотрудника не больше одной зарплаты на дату | `UNIQUE (employee_id, effective_date)` |
| текущая зарплата | последняя запись с `effective_date <= сегодня` |
| конец отпуска не раньше начала | `CHECK (end_date >= start_date)` |
| при удалении сотрудника уходят его зарплаты, заявки и история | `ON DELETE CASCADE` (сам сотрудник не удаляется, это защита целостности) |

## Индексы

`idx_users_role_id`, `idx_employees_department_id`, `idx_employees_position_id`, `idx_employees_status`, `idx_employees_name (last_name, first_name)`, `idx_salaries_employee_id`, `idx_time_off_employee_id`, `idx_time_off_status`, `idx_history_employee_id`. Первичные ключи и `UNIQUE` индексируются автоматически.
