-- big_categories table default data
INSERT INTO big_categories
  (id, category_name, transaction_type)
VALUES
  (1,  "収入", "income"),
  (2, "食費", "expense"),
  (3, "日用品", "expense"),
  (4, "趣味・娯楽", "expense"),
  (5, "交際費", "expense"),
  (6, "交通費", "expense"),
  (7, "衣服・美容", "expense"),
  (8, "健康・医療", "expense"),
  (9, "通信費", "expense"),
  (10, "教養・教育", "expense"),
  (11, "住宅", "expense"),
  (12, "水道・光熱費", "expense"),
  (13, "自動車", "expense"),
  (14, "保険", "expense"),
  (15, "税金・社会保険", "expense"),
  (16, "現金・カード", "expense"),
  (17, "その他", "expense");

-- medium_categories table default data
INSERT INTO medium_categories
  (id, category_name, big_category_id)
VALUES
  (1, "給与", 1),
  (2, "賞与", 1),
  (3, "一時所得", 1),
  (4, "事業所得", 1),
  (5, "その他収入", 1),
  (6, "食料品", 2),
  (7, "朝食", 2),
  (8, "昼食", 2),
  (9, "夕食", 2),
  (10, "外食", 2),
  (11, "カフェ", 2),
  (12, "その他食費", 2),
  (13, "消耗品", 3),
  (14, "子育て用品", 3),
  (15, "ペット用品", 3),
  (16, "家具", 3),
  (17, "家電", 3),
  (18, "その他日用品", 3),
  (19, "アウトドア", 4),
  (20, "旅行", 4),
  (21, "イベント", 4),
  (22, "スポーツ", 4),
  (23, "映画・動画", 4),
  (24, "音楽", 4),
  (25, "漫画", 4),
  (26, "書籍", 4),
  (27, "ゲーム", 4),
  (28, "その他趣味・娯楽", 4),
  (29, "飲み会", 5),
  (30, "プレゼント", 5),
  (31, "冠婚葬祭", 5),
  (32, "その他交際費", 5),
  (33, "電車", 6),
  (34, "バス", 6),
  (35, "タクシー", 6),
  (36, "新幹線", 6),
  (37, "飛行機", 6),
  (38, "その他交通費", 6),
  (39, "衣服", 7),
  (40, "アクセサリー", 7),
  (41, "クリーニング", 7),
  (42, "美容院・理髪", 7),
  (43, "化粧品", 7),
  (44, "エステ・ネイル", 7),
  (45, "その他衣服・美容", 7),
  (46, "病院", 8),
  (47, "薬", 8),
  (48, "ボディケア", 8),
  (49, "フィットネス", 8),
  (50, "その他健康・医療", 8),
  (51, "携帯電話", 9),
  (52, "固定電話", 9),
  (53, "インターネット", 9),
  (54, "放送サービス", 9),
  (55, "情報サービス", 9),
  (56, "宅配・運送", 9),
  (57, "切手・はがき", 9),
  (58, "その他通信費", 9),
  (59, "新聞", 10),
  (60, "参考書", 10),
  (61, "受験料", 10),
  (62, "学費", 10),
  (63, "習い事", 10),
  (64, "塾", 10),
  (65, "その他教養・教育", 10),
  (66, "家賃", 11),
  (67, "住宅ローン", 11),
  (68, "リフォーム", 11),
  (69, "その他住宅", 11),
  (70, "水道", 12),
  (71, "電気", 12),
  (72, "ガス", 12),
  (73, "その他水道・光熱費", 12),
  (74, "自動車ローン", 13),
  (75, "ガソリン", 13),
  (76, "駐車場", 13),
  (77, "高速料金", 13),
  (78, "車検・整備", 13),
  (79, "その他自動車", 13),
  (80, "生命保険", 14),
  (81, "医療保険", 14),
  (82, "自動車保険", 14),
  (83, "住宅保険", 14),
  (84, "学資保険", 14),
  (85, "その他保険", 14),
  (86, "所得税", 15),
  (87, "住民税", 15),
  (88, "年金保険料", 15),
  (89, "自動車税", 15),
  (90, "その他税金・社会保険", 15),
  (91, "現金引き出し", 16),
  (92, "カード引き落とし", 16),
  (93, "電子マネー", 16),
  (94, "立替金", 16),
  (95, "その他現金・カード", 16),
  (96, "仕送り", 17),
  (97, "お小遣い", 17),
  (98, "使途不明金", 17),
  (99, "雑費", 17);