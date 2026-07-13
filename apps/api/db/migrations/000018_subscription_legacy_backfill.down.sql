DELETE FROM store_subscriptions WHERE plan_id = (SELECT id FROM subscription_plans WHERE code = 'premium-contributor');
DELETE FROM subscription_plans WHERE code = 'premium-contributor';
