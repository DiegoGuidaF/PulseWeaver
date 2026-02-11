CREATE VIEW address_with_status AS
SELECT a.id,
       a.device_id,
       a.ip,
       a.created_at,
       COALESCE(s.latest_created_at, a.created_at) as updated_at,
       COALESCE(s.status, 1)                       as status
FROM addresses a
         LEFT JOIN (SELECT address_id, status, MAX(created_at) as latest_created_at
                    FROM address_status
                    GROUP BY address_id) s ON a.id = s.address_id;
