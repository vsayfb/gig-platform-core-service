DROP TABLE IF EXISTS reviews;
DROP TYPE  IF EXISTS review_role_context;
 

DROP TABLE IF EXISTS contracts;
DROP TYPE  IF EXISTS contract_status;
 

DROP TABLE IF EXISTS applications;
DROP TYPE  IF EXISTS application_status;
 

DROP TABLE IF EXISTS user_reputations;
 

DROP TABLE IF EXISTS user_locations;
 

DROP TABLE IF EXISTS gig_categories;
DROP TABLE IF EXISTS gig_locations;
DROP TABLE IF EXISTS gig_details;
DROP TABLE IF EXISTS gigs;
DROP TYPE  IF EXISTS duration_type;
DROP TYPE  IF EXISTS gig_status;
 

DROP TABLE IF EXISTS gig_categories;
DROP TABLE IF EXISTS user_categories;
DROP TABLE IF EXISTS categories;
DROP EXTENSION IF EXISTS vector;
 
DROP TABLE IF EXISTS user_auth;
DROP TABLE IF EXISTS users;
DROP EXTENSION IF EXISTS "uuid-ossp";
DROP EXTENSION IF EXISTS postgis;