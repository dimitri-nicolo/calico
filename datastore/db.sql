CREATE DATABASE IF NOT EXISTS tigera_backoffice CHARACTER SET utf8 COLLATE utf8_general_ci;
USE tigera_backoffice;

CREATE TABLE companies
(
  id    INT AUTO_INCREMENT PRIMARY KEY,
  uuid  CHAR(36) NOT NULL,
  `key` VARCHAR(20) NOT NULL,
  name  NVARCHAR(100) NOT NULL,
  CONSTRAINT companies_uuid_uindex UNIQUE (uuid),
  CONSTRAINT companies_key_uindex UNIQUE (`key`)
) ENGINE = InnoDB;

CREATE TABLE licenses
(
  id         INT AUTO_INCREMENT PRIMARY KEY,
  company_id INT NOT NULL,
  jwt        VARCHAR(7000) NOT NULL,
  CONSTRAINT licenses_companies_id_fk FOREIGN KEY (company_id) REFERENCES companies (id)
) ENGINE = InnoDB;

CREATE INDEX licenses_company_id_index ON licenses (company_id);

