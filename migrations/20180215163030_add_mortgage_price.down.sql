ALTER TABLE MortgageDetails 
DROP PRIMARY KEY, 
DROP COLUMN mortgagePrice,
ADD COLUMN id int(11) PRIMARY KEY;