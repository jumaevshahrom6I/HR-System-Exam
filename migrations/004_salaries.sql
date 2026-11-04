Begin;

create table if not exists salaries(
    id serial PRIMARY KEY,
    employee_id int not null REFERENCES employees(id) on delete CASCADE,
    amount numeric(12,2) not null check(amount > 0),
    effective_date date not null,
    created_at TIMESTAMPTZ not null  default now(),
    UNIQUE(employee_id, effective_date)
);

Create index if not exists idx_salaries_employee_id on salaries(employee_id);

Commit;