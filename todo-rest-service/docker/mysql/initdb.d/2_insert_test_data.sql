-- todo_list table test data
INSERT INTO todo_list
  (id, implementation_date, due_date, todo_content, complete_flag, user_id)
VALUES
  (1, "2020-07-05", "2020-07-05", "今月の予算を立てる", true, "taira"),
  (2, "2020-07-09", "2020-07-10", "コストコ鶏肉セール 5パック購入", true, "taira"),
  (3, "2020-07-10", "2020-07-10", "電車定期券更新", true, "taira"),
  (4, "2020-07-10", "2020-07-12", "醤油購入", false, "taira"),
  (5, "2020-08-01", "2020-08-10", "水道代支払い", false, "taira"),
  (6, "2020-08-01", "2020-08-15", "国保支払い", false, "taira"),
  (7, "2020-07-20", "2020-07-20", "給料日 飲みに行く", true , "anraku"),
  (8, "2020-07-21", "2020-07-22", "牛乳購入", false , "anraku"),
  (9, "2020-07-25", "2020-07-25", "自分用におむつ購入", false , "anraku"),
  (10, "2020-07-27", "2020-07-30", "牛肉2パック購入", false , "anraku");

-- group_todo_list table test data
INSERT INTO group_todo_list
  (id, implementation_date, due_date, todo_content, complete_flag, user_id, group_id)
VALUES
  (1, "2020-07-05", "2020-07-05", "今月の予算を立てる", true, "taira", 4),
  (2, "2020-07-09", "2020-07-10", "コストコ鶏肉セール 5パック購入", true, "taira", 4),
  (3, "2020-07-10", "2020-07-12", "醤油購入", false, "taira", 4),
  (4, "2020-07-20", "2020-07-20", "給料日 みんなで飲みに行く", true , "anraku", 4),
  (5, "2020-07-21", "2020-07-22", "牛乳購入", false , "anraku", 4),
  (6, "2020-07-27", "2020-07-30", "牛肉2パック購入", false , "anraku", 4);

-- group_tasks_users table test data
INSERT INTO group_tasks_users
  (id, user_id, group_id)
VALUES
  (1, "taira", 4),
  (2, "anraku", 4),
  (3, "furusawa", 4),
  (4, "ito", 4);

-- group_tasks table test data
INSERT INTO group_tasks
  (id, base_date, cycle_type, cycle, task_name, group_id, group_tasks_users_id)
VALUES
  (1, "2020-08-17", "every", 1, "料理", 4, 1),
  (2, "2020-08-10", "every", 3, "洗濯", 4, 4),
  (3, "2020-08-05", "every", 7, "トイレ掃除", 4, 3),
  (4, null, null, null, "台所掃除", 4, null),
  (5, "2020-08-17", "consecutive", 7, "風呂掃除", 4, 1),
  (6, "2020-08-20", "none", 1, "自分の部屋掃除", 4, 3),
  (7, "2020-08-15", "none", 1, "食料品買物", 4, 3);
