-- Reference catalog (not demo data): the interests users can pick from.
INSERT INTO interests (slug, label) VALUES
  ('travel', 'Travel'), ('hiking', 'Hiking'), ('cooking', 'Cooking'), ('coffee', 'Coffee'),
  ('wine', 'Wine'), ('music', 'Music'), ('concerts', 'Concerts'), ('cinema', 'Cinema'),
  ('series', 'TV series'), ('reading', 'Reading'), ('art', 'Art'), ('photography', 'Photography'),
  ('fitness', 'Fitness'), ('yoga', 'Yoga'), ('running', 'Running'), ('cycling', 'Cycling'),
  ('swimming', 'Swimming'), ('climbing', 'Climbing'), ('football', 'Football'), ('tennis', 'Tennis'),
  ('gaming', 'Video games'), ('board-games', 'Board games'), ('tech', 'Tech'), ('science', 'Science'),
  ('nature', 'Nature'), ('animals', 'Animals'), ('gardening', 'Gardening'), ('fashion', 'Fashion'),
  ('dancing', 'Dancing'), ('theatre', 'Theatre'), ('volunteering', 'Volunteering'), ('languages', 'Languages'),
  ('foodie', 'Foodie'), ('road-trips', 'Road trips'), ('festivals', 'Festivals'), ('meditation', 'Meditation')
ON CONFLICT (slug) DO NOTHING;
