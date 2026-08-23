DROP TEMPORARY TABLE IF EXISTS asukrtinform_3.company_file;

CREATE TEMPORARY TABLE asukrtinform_3.company_file (
  `hash` VARCHAR(255) NOT NULL,
  `inn` BIGINT NOT NULL,
  PRIMARY KEY (`inn`),
  KEY `idx_hash` (`hash`)
) ENGINE=MEMORY DEFAULT CHARSET=utf8mb4;

LOAD DATA INFILE '/var/lib/mysql-files/companies.csv'
INTO TABLE asukrtinform_3.company_file
CHARACTER SET utf8mb4
FIELDS TERMINATED BY ',' ENCLOSED BY '"' 
IGNORE 1 LINES 
(@idx, `hash`, @parent, @name, @okpo, `inn`);

call asukrtinform_3.export_csv();
