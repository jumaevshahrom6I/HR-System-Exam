Begin;

create table if not exists time_off(
    id serial PRIMARY Key,
    employee_id int not null REFERENCES employees(id) on delete cascade,
    approved_by int null REFERENCES users(id),
    leave_type varchar(20) not null check (leave_type in('vacation', 'sick')),
    start_date date not null,
    end_date date not null check(end_date >= start_date),
    status varchar(20) not null default 'pending' check (status in('pending', 'approved', 'rejected')),
    created_at TIMESTAMPTZ not null DEFAULT now(),
    updated_at TIMESTAMPTZ not null DEFAULT now()
);

Create index if not exists idx_time_off_employee_id on time_off(employee_id);
Create index if not exists idx_time_off_status on time_off(status);

Commit;