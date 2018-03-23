CREATE DATABASE IF NOT EXISTS tigera_backoffice CHARACTER SET utf8 COLLATE utf8_general_ci;
USE tigera_backoffice;

CREATE TABLE companies
(
  id    INT AUTO_INCREMENT PRIMARY KEY,
  name  NVARCHAR(100) NOT NULL,
  CONSTRAINT companies_name_uindex UNIQUE (name)
) ENGINE = InnoDB;

CREATE TABLE licenses
(
  id           INT AUTO_INCREMENT PRIMARY KEY,
  license_uuid CHAR(36) NOT NULL,
  nodes        INT,
  company_id   INT NOT NULL,
  cluster_guid VARCHAR(36),
  version      VARCHAR(100) NOT NULL,
  features     VARCHAR(100) NOT NULL,
  grace_period INT NOT NULL,
  checkin_int  INT,
  expiry       DATE NOT NULL,
  issued_at    DATE NOT NULL,
  jwt          VARCHAR(7000) NOT NULL,

  CONSTRAINT licenses_company_id_fk FOREIGN KEY (company_id) REFERENCES companies (id)
) ENGINE = InnoDB;

CREATE INDEX licenses_company_id_index ON licenses (company_id, id);
