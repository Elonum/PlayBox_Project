-- Добавление поля image_url
ALTER TABLE products
  ADD COLUMN IF NOT EXISTS image_url TEXT;

-- 1) LEGO Harry Potter 76414
WITH ins AS (
  INSERT INTO products
    (name, price, color, width_cm, height_cm, weight_g, description, quantity_in_stock, image_url)
  VALUES
    (
      'LEGO Harry Potter 76414 «Экспекто патронум»',
      7547.00,
      'голубой',
      26,
      38,
      970,
      'LEGO Harry Potter 76414 «Экспекто патронум» — набор, который позволяет воссоздать сцену из мира Гарри Поттера, где герой использует заклинание «Экспекто патронум», чтобы вызвать своего волшебного покровителя и защититься от дементоров.',
      300,
      'https://drive.google.com/uc?export=download&id=1_fGwu0cbxS1x6czUpVa8COaI_UpXi3Bf'
    )
  RETURNING product_id
)
INSERT INTO products_categories (product_id, category_id)
SELECT
  ins.product_id,
  c.category_id
FROM ins
CROSS JOIN LATERAL (
  VALUES
    ((SELECT category_id FROM categories WHERE name = 'Конструкторы')),
    ((SELECT category_id FROM categories WHERE name = 'Развивающие наборы'))
) AS c(category_id);

-- 2) Плюшевый мишка Pangcangshu
WITH ins AS (
  INSERT INTO products
    (name, price, color, width_cm, height_cm, weight_g, description, quantity_in_stock, image_url)
  VALUES
    (
      'Плюшевый мишка Pangcangshu',
      3108.00,
      'розовый',
      20,
      65,
      500,
      NULL,
      550,
      'https://drive.google.com/uc?export=download&id=13iDD4U-yvCDmli9yDLeUVkiBSk6Sp2wM'
    )
  RETURNING product_id
)
INSERT INTO products_categories (product_id, category_id)
SELECT
  ins.product_id,
  (SELECT category_id FROM categories WHERE name = 'Мягкие игрушки')
FROM ins;

-- 3) Набор доктора RABBIT (13 предм.)
WITH ins AS (
  INSERT INTO products
    (name, price, color, width_cm, height_cm, weight_g, description, quantity_in_stock, image_url)
  VALUES
    (
      'Набор доктора RABBIT (13 предм.)',
      1063.00,
      'розовый',
      19,
      17,
      400,
      'Набор доктора RABBIT (13 предметов) на батарейках в розовом чемоданчике – это замечательный подарок для девочек в возрасте от 7 до 14 лет, которые любят играть в доктора. Набор включает в себя 13 предметов, выполненных из качественного пластика, что гарантирует их долговечность и безопасность для детей. Розовый цвет делает набор ярким и привлекательным для девочек.',
      150,
      'https://drive.google.com/uc?export=download&id=1Qxpbcm8-v8i1xYJfoJkLXN1LbGMYo0P1'
    )
  RETURNING product_id
)
INSERT INTO products_categories (product_id, category_id)
SELECT
  ins.product_id,
  (SELECT category_id FROM categories WHERE name = 'Ролевые игры')
FROM ins;
