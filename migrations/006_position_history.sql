Begin;

create table if not exists employee_position_history(
    id serial primary key,
    employee_id int not null REFERENCES employees(id) on delete CASCADE,
    department_id int not null REFERENCES departments(id),
    position_id int not null REFERENCES positions(id),
    start_date date not NULL,
    end_date date NULL check(end_date >= start_date),
    created_at timestamptz not null DEFAULT now()
);

Create index if not exists idx_history_employee_id on employee_position_history(employee_id);

Commit;