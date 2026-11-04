create table if not exists departments(
    id serial PRIMARY KEY,
    name varchar(100) not null UNIQUE,
    created_at timestamptz not null DEFAULT now(),     
    updated_at timestamptz not null DEFAULT now()
);

create table if not exists positions(
    id serial PRIMARY KEY,
    name VARCHAR(100) not null UNIQUE,
    base_salary NUMERIC(12,2) not NULL check(base_salary >= 0),
    created_at timestamptz not null default now(), 
    updated_at timestamptz not null default now()
);