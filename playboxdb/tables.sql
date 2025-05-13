-- 1. Справочники / родительские таблицы

-- 1.1 Роли пользователей
CREATE TABLE roles (
    role_id    SERIAL PRIMARY KEY,
    role_name  VARCHAR(50) UNIQUE NOT NULL
);

-- 1.2 Категории товаров
CREATE TABLE categories (
    category_id       SERIAL PRIMARY KEY,
    name              VARCHAR(100) UNIQUE NOT NULL
);

-- 1.3 Статусы заказов
CREATE TABLE order_statuses (
    status_id         SERIAL PRIMARY KEY,
    status_name       VARCHAR(50) UNIQUE NOT NULL
);

-- 1.4 Страны
CREATE TABLE countries (
    country_id        SERIAL PRIMARY KEY,
    country_name      VARCHAR(100) UNIQUE NOT NULL
);

-- 2. Основные сущности

-- 2.1 Пользователи
CREATE TABLE users (
    user_id             SERIAL PRIMARY KEY,
    role_id             INTEGER NOT NULL REFERENCES roles(role_id),
    first_name          VARCHAR(50)   NOT NULL,
    last_name           VARCHAR(50)   NOT NULL,
    email               VARCHAR(150)  UNIQUE NOT NULL,
    phone               VARCHAR(20)   UNIQUE NOT NULL,
    password_hash       VARCHAR(256)  NOT NULL,
    profile_picture_url TEXT          NULL,
    registration_ts     TIMESTAMPTZ   NOT NULL DEFAULT now()
);

-- 2.2 Товары
CREATE TABLE products (
    product_id        SERIAL PRIMARY KEY,
    name              VARCHAR(150)  UNIQUE NOT NULL,
    price             NUMERIC(10,2) NOT NULL,
    color             VARCHAR(50)   NOT NULL,
    width_cm          INTEGER       NOT NULL CHECK (width_cm > 0),
    height_cm         INTEGER       NOT NULL CHECK (height_cm > 0),
    weight_g          INTEGER       NOT NULL CHECK (weight_g > 0),
    description       TEXT          NULL,
    quantity_in_stock INTEGER       NOT NULL CHECK (quantity_in_stock >= 0)
);

-- 2.3 Склады
CREATE TABLE warehouses (
    warehouse_id      SERIAL PRIMARY KEY,
    name              VARCHAR(100) UNIQUE NOT NULL,
    capacity          INTEGER       NOT NULL,
    country_id        INTEGER NOT NULL REFERENCES countries(country_id)
);

-- 2.4 Производственные линии
CREATE TABLE work_lines (
    line_id           SERIAL PRIMARY KEY,
    name              VARCHAR(50) UNIQUE NOT NULL,
    country_id        INTEGER NOT NULL REFERENCES countries(country_id)
);

-- 2.5 Рабочий персонал
CREATE TABLE staff (
    staff_id          SERIAL PRIMARY KEY,
    name              VARCHAR(50) NOT NULL,
    surname           VARCHAR(50),
    patronymic        VARCHAR(50),
    age               INTEGER NOT NULL,
    position          VARCHAR(100) NOT NULL,
    experience_years  INTEGER NOT NULL,
    salary            MONEY
);

-- 3. Зависимые таблицы

-- 3.1 Связь товары ↔ категории (M:N)
CREATE TABLE products_categories (
    product_id        INTEGER NOT NULL REFERENCES products(product_id),
    category_id       INTEGER NOT NULL REFERENCES categories(category_id),
    PRIMARY KEY (product_id, category_id)
);

-- 3.2 Корзина
CREATE TABLE carts (
    cart_id           SERIAL PRIMARY KEY,
    user_id           INTEGER NOT NULL REFERENCES users(user_id)
);

CREATE TABLE cart_items (
    cart_item_id      SERIAL PRIMARY KEY,
    cart_id           INTEGER NOT NULL REFERENCES carts(cart_id) ON DELETE CASCADE,
    product_id        INTEGER NOT NULL REFERENCES products(product_id),
    quantity          INTEGER NOT NULL CHECK (quantity > 0),
    UNIQUE (cart_id, product_id)
);

-- 3.3 Заказы
CREATE TABLE orders (
    order_id          SERIAL PRIMARY KEY,
    user_id           INTEGER NOT NULL REFERENCES users(user_id),
    status_id         INTEGER NOT NULL REFERENCES order_statuses(status_id),
    order_ts          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE order_items (
    order_item_id     SERIAL PRIMARY KEY,
    order_id          INTEGER NOT NULL REFERENCES orders(order_id) ON DELETE CASCADE,
    product_id        INTEGER NOT NULL REFERENCES products(product_id),
    quantity          INTEGER NOT NULL CHECK (quantity > 0)
);

-- 3.4 Платёжные карты
CREATE TABLE payment_cards (
    card_id         SERIAL PRIMARY KEY,
    user_id         INTEGER     NOT NULL REFERENCES users(user_id),
    cardholder_name VARCHAR(100) NOT NULL,
    card_number     VARCHAR(19)  NOT NULL,
    exp_month       SMALLINT    NOT NULL CHECK (exp_month BETWEEN 1 AND 12),
    exp_year        SMALLINT    NOT NULL
);

-- 3.5 Запасы на складах
CREATE TABLE warehouse_products (
    warehouse_id      INTEGER NOT NULL REFERENCES warehouses(warehouse_id),
    product_id        INTEGER NOT NULL REFERENCES products(product_id),
    quantity          INTEGER NOT NULL CHECK (quantity >= 0),
    PRIMARY KEY (warehouse_id, product_id)
);

-- 3.6 Связь персонал ↔ линии
CREATE TABLE staff_worklines (
    staff_id          INTEGER NOT NULL REFERENCES staff(staff_id),
    line_id           INTEGER NOT NULL REFERENCES work_lines(line_id),
    PRIMARY KEY (staff_id, line_id)
);


-- 4. Заполнение справочных таблиц

-- 4.1 Роли
INSERT INTO roles (role_name) VALUES 
  ('admin'),
  ('user');

-- 4.2 Категории
INSERT INTO categories (name) VALUES
  ('Конструкторы'),
  ('Куклы и аксессуары'),
  ('Мягкие игрушки'),
  ('Настольные игры'),
  ('Развивающие наборы'),
  ('Ролевые игры'),
  ('Спортивный инвентарь'),
  ('Творческие наборы'),
  ('Транспортные игрушки'),
  ('Электронные игрушки');

-- 4.3 Статусы заказов
INSERT INTO order_statuses (status_name) VALUES
  ('Новый'),
  ('Подтверждён'),
  ('В обработке'),
  ('Отправлен'),
  ('Доставлен'),
  ('Отменён');

-- 4.4 Страны
INSERT INTO countries (country_name) VALUES
  ('Россия'),
  ('Беларусь'),
  ('Казахстан');
