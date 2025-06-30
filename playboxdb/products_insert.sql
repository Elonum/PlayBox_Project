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
      'https://msk.toys/upload/iblock/bb3/4oawnrhmpqs38j5j32ovf00ehswhbjuk.jpg'
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
      'https://ae04.alicdn.com/kf/Hc6d29335b9b64f3e8277e1cd7be01912A.jpg'
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
      'https://cdn1.ozone.ru/s3/multimedia-2/c600/6728324402.jpg'
    )
  RETURNING product_id
)
INSERT INTO products_categories (product_id, category_id)
SELECT
  ins.product_id,
  (SELECT category_id FROM categories WHERE name = 'Ролевые игры')
FROM ins;

-- 4) Кукла Mattel Monster High
WITH ins AS (
  INSERT INTO products
    (name, price, color, width_cm, height_cm, weight_g, description, quantity_in_stock, image_url)
  VALUES
    (
      'Кукла Mattel Monster High HYV90',
      2694.00,
      'разноцветный',
      5,
      28,
      450,
      'Коллекционная кукла Monster High Operetta — уникальное издание создано для любителей стилей рокабилли и ретро. Оперетта, дочь легендарного Призрака Оперы, возвращается во всей своей красе. Эта необычная кукла восхищает изысканными деталями, отсылающими к эстетике пятидесятых годов, — ее характерным стилем, сочетающим винтажную элегантность с темной, жуткой атмосферой.',
      50,
      'https://dollsfest.ru/pictures/product/big/10213_big.jpg'
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
    ((SELECT category_id FROM categories WHERE name = 'Куклы и аксессуары')),
    ((SELECT category_id FROM categories WHERE name = 'Ролевые игры'))
) AS c(category_id);
