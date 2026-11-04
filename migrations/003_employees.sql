Begin;

create table if not exists employees(
    id serial PRIMARY KEY,
    user_id INT not null UNIQUE REFERENCES users(id),
    department_id int not null REFERENCES departments(id) on delete RESTRICT,
    position_id int not null REFERENCES positions(id) on delete RESTRICT,
    first_name varchar(50) not null,
    last_name varchar(50) not null,
    phone VARCHAR(20) not null,
    hire_date date not null,
    termination_date date,
    constraint employees_dates
        check(termination_date is null or termination_date >= hire_date),
    status varchar(20) not null default 'active'
        check(status in('active', 'on_leave', 'terminated')),
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);

Create index if not exists idx_employees_department_id on employees(department_id);
Create index if not exists idx_employees_position_id on employees(position_id);
Create index if not exists idx_employees_status on employees(status);
Create index if not exists idx_employees_name on employees(last_name, first_name);

Commit;