-- Dummy data seed for the MOH Clinician App
-- Run this after the schema script, while connected to the target database.
--
-- This script clears existing application data in clinician_app and inserts
-- a Uganda referral-hospital-oriented testing dataset with working user accounts.
--
-- Demo logins:
--   National admin:
--     admin                      / admin
--   Facility admin:
--     admin-region               / test
--   Clinician:
--     lillian.anzima@demo.test   / test
--   Additional demo users:
--     all other demo users       / test
--
-- Passwords are stored as SHA-1 hashes because that is what the current app expects.

TRUNCATE TABLE
    clinician_app.attendance_records,
    clinician_app.surgeries,
    clinician_app.ward_rounds,
    clinician_app.investigations,
    clinician_app.weeklyreport,
    clinician_app.staffleave,
    clinician_app.users,
    clinician_app.employeerights,
    clinician_app.targets,
    clinician_app.indicators,
    clinician_app.department_roles,
    clinician_app.employees,
    clinician_app.rights,
    clinician_app.leavetypes,
    clinician_app.specialist_titles,
    clinician_app.departments,
    clinician_app.facilities,
    clinician_app.lg
RESTART IDENTITY CASCADE;

INSERT INTO clinician_app.lg (id, lg_name, lg_type) VALUES
    (1, 'Arua District', 'District'),
    (2, 'Wakiso District', 'District'),
    (3, 'Kabarole District', 'District'),
    (4, 'Gulu District', 'District'),
    (5, 'Hoima District', 'District'),
    (6, 'Jinja District', 'District'),
    (7, 'Kabale District', 'District'),
    (8, 'Kayunga District', 'District'),
    (9, 'Lira District', 'District'),
    (10, 'Masaka District', 'District'),
    (11, 'Mbale District', 'District'),
    (12, 'Mbarara District', 'District'),
    (13, 'Moroto District', 'District'),
    (14, 'Mubende District', 'District'),
    (15, 'Soroti District', 'District'),
    (16, 'Kampala Capital City', 'City');

INSERT INTO clinician_app.facilities (id, f_name, f_level, f_lg, created_by, created_on) VALUES
    (1, 'Arua Regional Referral Hospital', 'Regional Referral', 1, NULL, NOW()),
    (2, 'Entebbe Regional Referral Hospital', 'Regional Referral', 2, NULL, NOW()),
    (3, 'Fort Portal Regional Referral Hospital', 'Regional Referral', 3, NULL, NOW()),
    (4, 'Gulu Regional Referral Hospital', 'Regional Referral', 4, NULL, NOW()),
    (5, 'Hoima Regional Referral Hospital', 'Regional Referral', 5, NULL, NOW()),
    (6, 'Jinja Regional Referral Hospital', 'Regional Referral', 6, NULL, NOW()),
    (7, 'Kabale Regional Referral Hospital', 'Regional Referral', 7, NULL, NOW()),
    (8, 'Kayunga Regional Referral Hospital', 'Regional Referral', 8, NULL, NOW()),
    (9, 'Lira Regional Referral Hospital', 'Regional Referral', 9, NULL, NOW()),
    (10, 'Masaka Regional Referral Hospital', 'Regional Referral', 10, NULL, NOW()),
    (11, 'Mbale Regional Referral Hospital', 'Regional Referral', 11, NULL, NOW()),
    (12, 'Mbarara Regional Referral Hospital', 'Regional Referral', 12, NULL, NOW()),
    (13, 'Moroto Regional Referral Hospital', 'Regional Referral', 13, NULL, NOW()),
    (14, 'Mubende Regional Referral Hospital', 'Regional Referral', 14, NULL, NOW()),
    (15, 'Soroti Regional Referral Hospital', 'Regional Referral', 15, NULL, NOW()),
    (16, 'China-Uganda Friendship Hospital, Naguru', 'Specialized Referral', 16, NULL, NOW());

INSERT INTO clinician_app.departments (id, d_name) VALUES
    (1, 'Surgery'),
    (2, 'Internal Medicine'),
    (3, 'Paediatrics'),
    (4, 'Obstetrics and Gynaecology');

INSERT INTO clinician_app.specialist_titles (id, title) VALUES
    (1, 'Medical Officer(SG)'),
    (2, 'Medical Officer'),
    (3, 'Medical Officer(Specialist)'),
    (4, 'Senior Consultant'),
    (5, 'Consultant'),
    (6, 'Senior Nursing Officer'),
    (7, 'Nursing Officer');

-- Application roles required by this build:
--   admin    = national administrator
--   approver = facility-level administrator / manager
--   user     = clinician / medical officer
INSERT INTO clinician_app.rights (id, rights) VALUES
    (1, 'admin'),
    (2, 'user'),
    (3, 'approver');

INSERT INTO clinician_app.leavetypes (leave_type_id, leave_type_name, description, created_at, updated_at) VALUES
    (1, 'Annual Leave', 'Annual leave', NOW(), NOW()),
    (2, 'Sick Leave', 'Sick leave', NOW(), NOW()),
    (3, 'Maternity Leave', 'Maternity leave', NOW(), NOW()),
    (4, 'Paternity Leave', 'Paternity leave', NOW(), NOW()),
    (5, 'Bereavement Leave', 'Bereavement leave', NOW(), NOW()),
    (6, 'Unpaid Leave', 'Unpaid leave', NOW(), NOW()),
    (7, 'Study Leave', 'Study leave', NOW(), NOW()),
    (8, 'Field Activities Leave', 'Field activities leave', NOW(), NOW()),
    (9, 'Emergency Leave', 'Emergency leave', NOW(), NOW());

INSERT INTO clinician_app.employees (
    id, fname, lname, oname, specialisation, department, facility, created_by, created_on, title
) VALUES
    (1001, 'Sarah', 'Adriko', NULL, 'Medical Superintendent', 2, 1, NULL, NOW(), 4),
    (1002, 'Lillian', 'Anzima', NULL, 'General Surgery', 1, 1, 1001, NOW(), 3),
    (1003, 'Godfrey', 'Dradriga', NULL, 'Internal Medicine', 2, 1, 1001, NOW(), 3),
    (1004, 'Racheal', 'Anguzu', NULL, 'Paediatrics', 3, 1, 1001, NOW(), 3),
    (1005, 'Mariam', 'Avako', NULL, 'Obstetrics and Gynaecology', 4, 1, 1001, NOW(), 3),
    (1006, 'Moses', 'Kato', NULL, 'Medical Superintendent', 2, 2, 1001, NOW(), 5),
    (1007, 'Allan', 'Ssembatya', NULL, 'General Surgery', 1, 2, 1006, NOW(), 3),
    (1008, 'Juliet', 'Nalwadda', NULL, 'Internal Medicine', 2, 2, 1006, NOW(), 3),
    (1009, 'Mark', 'Nsubuga', NULL, 'Paediatrics', 3, 2, 1006, NOW(), 3),
    (1010, 'Allen', 'Nakato', NULL, 'Obstetrics and Gynaecology', 4, 2, 1006, NOW(), 3),
    (1011, 'Harriet', 'Ayesiga', NULL, 'Medical Superintendent', 2, 3, 1001, NOW(), 5),
    (1012, 'Christine', 'Asiimwe', NULL, 'General Surgery', 1, 3, 1011, NOW(), 3),
    (1013, 'Michael', 'Kiconco', NULL, 'Internal Medicine', 2, 3, 1011, NOW(), 3),
    (1014, 'Peace', 'Komugisha', NULL, 'Paediatrics', 3, 3, 1011, NOW(), 3),
    (1015, 'Joyce', 'Kembabazi', NULL, 'Obstetrics and Gynaecology', 4, 3, 1011, NOW(), 3),
    (1016, 'Patrick', 'Ocaya', NULL, 'Medical Superintendent', 2, 4, 1001, NOW(), 5),
    (1017, 'Martin', 'Ojara', NULL, 'General Surgery', 1, 4, 1016, NOW(), 3),
    (1018, 'Beatrice', 'Lamwaka', NULL, 'Internal Medicine', 2, 4, 1016, NOW(), 3),
    (1019, 'Denis', 'Opio', NULL, 'Paediatrics', 3, 4, 1016, NOW(), 3),
    (1020, 'Rachel', 'Auma', NULL, 'Obstetrics and Gynaecology', 4, 4, 1016, NOW(), 3),
    (1021, 'James', 'Byaruhanga', NULL, 'Medical Superintendent', 2, 5, 1001, NOW(), 5),
    (1022, 'Paul', 'Byamukama', NULL, 'General Surgery', 1, 5, 1021, NOW(), 3),
    (1023, 'Andrew', 'Asiimwe', NULL, 'Internal Medicine', 2, 5, 1021, NOW(), 3),
    (1024, 'Immaculate', 'Asiimwe', NULL, 'Paediatrics', 3, 5, 1021, NOW(), 3),
    (1025, 'Joel', 'Muhumuza', NULL, 'Obstetrics and Gynaecology', 4, 5, 1021, NOW(), 3),
    (1026, 'Rebecca', 'Nandutu', NULL, 'Medical Superintendent', 2, 6, 1001, NOW(), 5),
    (1027, 'Diana', 'Nabulya', NULL, 'General Surgery', 1, 6, 1026, NOW(), 3),
    (1028, 'Noah', 'Waiswa', NULL, 'Internal Medicine', 2, 6, 1026, NOW(), 3),
    (1029, 'Shamim', 'Namusoke', NULL, 'Paediatrics', 3, 6, 1026, NOW(), 3),
    (1030, 'Prossy', 'Nabirye', NULL, 'Obstetrics and Gynaecology', 4, 6, 1026, NOW(), 3),
    (1031, 'David', 'Turyasingura', NULL, 'Medical Superintendent', 2, 7, 1001, NOW(), 5),
    (1032, 'Isaac', 'Turyatemba', NULL, 'General Surgery', 1, 7, 1031, NOW(), 3),
    (1033, 'Sarah', 'Kobusingye', NULL, 'Internal Medicine', 2, 7, 1031, NOW(), 3),
    (1034, 'Harriet', 'Natukunda', NULL, 'Paediatrics', 3, 7, 1031, NOW(), 3),
    (1035, 'Mercy', 'Twinomujuni', NULL, 'Obstetrics and Gynaecology', 4, 7, 1031, NOW(), 3),
    (1036, 'Irene', 'Namubiru', NULL, 'Medical Superintendent', 2, 8, 1001, NOW(), 5),
    (1037, 'Joan', 'Nakirya', NULL, 'General Surgery', 1, 8, 1036, NOW(), 3),
    (1038, 'Richard', 'Ssali', NULL, 'Internal Medicine', 2, 8, 1036, NOW(), 3),
    (1039, 'Moses', 'Bbosa', NULL, 'Paediatrics', 3, 8, 1036, NOW(), 3),
    (1040, 'Harold', 'Sserwanja', NULL, 'Obstetrics and Gynaecology', 4, 8, 1036, NOW(), 3),
    (1041, 'Samuel', 'Okello', NULL, 'Medical Superintendent', 2, 9, 1001, NOW(), 5),
    (1042, 'Charles', 'Obong', NULL, 'General Surgery', 1, 9, 1041, NOW(), 3),
    (1043, 'Peninah', 'Acan', NULL, 'Internal Medicine', 2, 9, 1041, NOW(), 3),
    (1044, 'Gloria', 'Atim', NULL, 'Paediatrics', 3, 9, 1041, NOW(), 3),
    (1045, 'Annet', 'Acam', NULL, 'Obstetrics and Gynaecology', 4, 9, 1041, NOW(), 3),
    (1046, 'Agnes', 'Nakato', NULL, 'Medical Superintendent', 2, 10, 1001, NOW(), 5),
    (1047, 'Esther', 'Ssekimpi', NULL, 'General Surgery', 1, 10, 1046, NOW(), 3),
    (1048, 'Robert', 'Kizito', NULL, 'Internal Medicine', 2, 10, 1046, NOW(), 3),
    (1049, 'Sharon', 'Nakimuli', NULL, 'Paediatrics', 3, 10, 1046, NOW(), 3),
    (1050, 'Rebecca', 'Birungi', NULL, 'Obstetrics and Gynaecology', 4, 10, 1046, NOW(), 3),
    (1051, 'Peter', 'Wekesa', NULL, 'Medical Superintendent', 2, 11, 1001, NOW(), 5),
    (1052, 'Ronald', 'Masaba', NULL, 'General Surgery', 1, 11, 1051, NOW(), 3),
    (1053, 'Mildred', 'Namatovu', NULL, 'Internal Medicine', 2, 11, 1051, NOW(), 3),
    (1054, 'Stephen', 'Wanyama', NULL, 'Paediatrics', 3, 11, 1051, NOW(), 3),
    (1055, 'Ivy', 'Namuddu', NULL, 'Obstetrics and Gynaecology', 4, 11, 1051, NOW(), 3),
    (1056, 'Grace', 'Tumusiime', NULL, 'Medical Superintendent', 2, 12, 1001, NOW(), 5),
    (1057, 'Patricia', 'Ampaire', NULL, 'General Surgery', 1, 12, 1056, NOW(), 3),
    (1058, 'Brian', 'Mugisha', NULL, 'Internal Medicine', 2, 12, 1056, NOW(), 3),
    (1059, 'Esther', 'Busingye', NULL, 'Paediatrics', 3, 12, 1056, NOW(), 3),
    (1060, 'Godfrey', 'Taremwa', NULL, 'Obstetrics and Gynaecology', 4, 12, 1056, NOW(), 3),
    (1061, 'Simon', 'Lokiru', NULL, 'Medical Superintendent', 2, 13, 1001, NOW(), 5),
    (1062, 'Emmanuel', 'Longole', NULL, 'General Surgery', 1, 13, 1061, NOW(), 3),
    (1063, 'Doreen', 'Aciro', NULL, 'Internal Medicine', 2, 13, 1061, NOW(), 3),
    (1064, 'Rose', 'Akuj', NULL, 'Paediatrics', 3, 13, 1061, NOW(), 3),
    (1065, 'Hilda', 'Loputia', NULL, 'Obstetrics and Gynaecology', 4, 13, 1061, NOW(), 3),
    (1066, 'Ruth', 'Namayanja', NULL, 'Medical Superintendent', 2, 14, 1001, NOW(), 5),
    (1067, 'Hellen', 'Namusoke', NULL, 'General Surgery', 1, 14, 1066, NOW(), 3),
    (1068, 'Simon', 'Kateregga', NULL, 'Internal Medicine', 2, 14, 1066, NOW(), 3),
    (1069, 'Lydia', 'Ssenyonga', NULL, 'Paediatrics', 3, 14, 1066, NOW(), 3),
    (1070, 'Brian', 'Kasule', NULL, 'Obstetrics and Gynaecology', 4, 14, 1066, NOW(), 3),
    (1071, 'Joseph', 'Ecom', NULL, 'Medical Superintendent', 2, 15, 1001, NOW(), 5),
    (1072, 'Vincent', 'Ekuwam', NULL, 'General Surgery', 1, 15, 1071, NOW(), 3),
    (1073, 'Janet', 'Elotu', NULL, 'Internal Medicine', 2, 15, 1071, NOW(), 3),
    (1074, 'Kenneth', 'Omolo', NULL, 'Paediatrics', 3, 15, 1071, NOW(), 3),
    (1075, 'Janet', 'Apio', NULL, 'Obstetrics and Gynaecology', 4, 15, 1071, NOW(), 3),
    (1076, 'Miriam', 'Nambuya', NULL, 'Medical Superintendent', 2, 16, 1001, NOW(), 5),
    (1077, 'Florah', 'Nandera', NULL, 'General Surgery', 1, 16, 1076, NOW(), 3),
    (1078, 'David', 'Wandera', NULL, 'Internal Medicine', 2, 16, 1076, NOW(), 3),
    (1079, 'Naomi', 'Nabwire', NULL, 'Paediatrics', 3, 16, 1076, NOW(), 3),
    (1080, 'Claire', 'Namutebi', NULL, 'Obstetrics and Gynaecology', 4, 16, 1076, NOW(), 3);

INSERT INTO clinician_app.employeerights (id, employee, rights) VALUES
    (1, 1001, 1),
    (2, 1002, 2),
    (3, 1003, 2),
    (4, 1004, 2),
    (5, 1005, 2),
    (6, 1006, 3),
    (7, 1007, 2),
    (8, 1008, 2),
    (9, 1009, 2),
    (10, 1010, 2),
    (11, 1011, 3),
    (12, 1012, 2),
    (13, 1013, 2),
    (14, 1014, 2),
    (15, 1015, 2),
    (16, 1016, 3),
    (17, 1017, 2),
    (18, 1018, 2),
    (19, 1019, 2),
    (20, 1020, 2),
    (21, 1021, 3),
    (22, 1022, 2),
    (23, 1023, 2),
    (24, 1024, 2),
    (25, 1025, 2),
    (26, 1026, 3),
    (27, 1027, 2),
    (28, 1028, 2),
    (29, 1029, 2),
    (30, 1030, 2),
    (31, 1031, 3),
    (32, 1032, 2),
    (33, 1033, 2),
    (34, 1034, 2),
    (35, 1035, 2),
    (36, 1036, 3),
    (37, 1037, 2),
    (38, 1038, 2),
    (39, 1039, 2),
    (40, 1040, 2),
    (41, 1041, 3),
    (42, 1042, 2),
    (43, 1043, 2),
    (44, 1044, 2),
    (45, 1045, 2),
    (46, 1046, 3),
    (47, 1047, 2),
    (48, 1048, 2),
    (49, 1049, 2),
    (50, 1050, 2),
    (51, 1051, 3),
    (52, 1052, 2),
    (53, 1053, 2),
    (54, 1054, 2),
    (55, 1055, 2),
    (56, 1056, 3),
    (57, 1057, 2),
    (58, 1058, 2),
    (59, 1059, 2),
    (60, 1060, 2),
    (61, 1061, 3),
    (62, 1062, 2),
    (63, 1063, 2),
    (64, 1064, 2),
    (65, 1065, 2),
    (66, 1066, 3),
    (67, 1067, 2),
    (68, 1068, 2),
    (69, 1069, 2),
    (70, 1070, 2),
    (71, 1071, 3),
    (72, 1072, 2),
    (73, 1073, 2),
    (74, 1074, 2),
    (75, 1075, 2),
    (76, 1076, 3),
    (77, 1077, 2),
    (78, 1078, 2),
    (79, 1079, 2),
    (80, 1080, 2);

INSERT INTO clinician_app.users (id, username, pssword, employees, created_by, created_on, rights, access_scope) VALUES
    (1, 'sarah.adriko@demo.test', 'a94a8fe5ccb19ba61c4c0873d391e987982fbbd3', 1001, NULL, NOW(), 'admin', 'national'),
    (2, 'lillian.anzima@demo.test', 'a94a8fe5ccb19ba61c4c0873d391e987982fbbd3', 1002, 1001, NOW(), 'user', 'individual'),
    (3, 'godfrey.dradriga@demo.test', 'a94a8fe5ccb19ba61c4c0873d391e987982fbbd3', 1003, 1001, NOW(), 'user', 'individual'),
    (4, 'racheal.anguzu@demo.test', 'a94a8fe5ccb19ba61c4c0873d391e987982fbbd3', 1004, 1001, NOW(), 'user', 'individual'),
    (5, 'mariam.avako@demo.test', 'a94a8fe5ccb19ba61c4c0873d391e987982fbbd3', 1005, 1001, NOW(), 'user', 'individual'),
    (6, 'moses.kato@demo.test', 'a94a8fe5ccb19ba61c4c0873d391e987982fbbd3', 1006, 1001, NOW(), 'approver', 'facility'),
    (7, 'allan.ssembatya@demo.test', 'a94a8fe5ccb19ba61c4c0873d391e987982fbbd3', 1007, 1006, NOW(), 'user', 'individual'),
    (8, 'juliet.nalwadda@demo.test', 'a94a8fe5ccb19ba61c4c0873d391e987982fbbd3', 1008, 1006, NOW(), 'user', 'individual'),
    (9, 'mark.nsubuga@demo.test', 'a94a8fe5ccb19ba61c4c0873d391e987982fbbd3', 1009, 1006, NOW(), 'user', 'individual'),
    (10, 'allen.nakato@demo.test', 'a94a8fe5ccb19ba61c4c0873d391e987982fbbd3', 1010, 1006, NOW(), 'user', 'individual'),
    (11, 'harriet.ayesiga@demo.test', 'a94a8fe5ccb19ba61c4c0873d391e987982fbbd3', 1011, 1001, NOW(), 'approver', 'facility'),
    (12, 'christine.asiimwe@demo.test', 'a94a8fe5ccb19ba61c4c0873d391e987982fbbd3', 1012, 1011, NOW(), 'user', 'individual'),
    (13, 'michael.kiconco@demo.test', 'a94a8fe5ccb19ba61c4c0873d391e987982fbbd3', 1013, 1011, NOW(), 'user', 'individual'),
    (14, 'peace.komugisha@demo.test', 'a94a8fe5ccb19ba61c4c0873d391e987982fbbd3', 1014, 1011, NOW(), 'user', 'individual'),
    (15, 'joyce.kembabazi@demo.test', 'a94a8fe5ccb19ba61c4c0873d391e987982fbbd3', 1015, 1011, NOW(), 'user', 'individual'),
    (16, 'patrick.ocaya@demo.test', 'a94a8fe5ccb19ba61c4c0873d391e987982fbbd3', 1016, 1001, NOW(), 'approver', 'facility'),
    (17, 'martin.ojara@demo.test', 'a94a8fe5ccb19ba61c4c0873d391e987982fbbd3', 1017, 1016, NOW(), 'user', 'individual'),
    (18, 'beatrice.lamwaka@demo.test', 'a94a8fe5ccb19ba61c4c0873d391e987982fbbd3', 1018, 1016, NOW(), 'user', 'individual'),
    (19, 'denis.opio@demo.test', 'a94a8fe5ccb19ba61c4c0873d391e987982fbbd3', 1019, 1016, NOW(), 'user', 'individual'),
    (20, 'rachel.auma@demo.test', 'a94a8fe5ccb19ba61c4c0873d391e987982fbbd3', 1020, 1016, NOW(), 'user', 'individual'),
    (21, 'james.byaruhanga@demo.test', 'a94a8fe5ccb19ba61c4c0873d391e987982fbbd3', 1021, 1001, NOW(), 'approver', 'facility'),
    (22, 'paul.byamukama@demo.test', 'a94a8fe5ccb19ba61c4c0873d391e987982fbbd3', 1022, 1021, NOW(), 'user', 'individual'),
    (23, 'andrew.asiimwe@demo.test', 'a94a8fe5ccb19ba61c4c0873d391e987982fbbd3', 1023, 1021, NOW(), 'user', 'individual'),
    (24, 'immaculate.asiimwe@demo.test', 'a94a8fe5ccb19ba61c4c0873d391e987982fbbd3', 1024, 1021, NOW(), 'user', 'individual'),
    (25, 'joel.muhumuza@demo.test', 'a94a8fe5ccb19ba61c4c0873d391e987982fbbd3', 1025, 1021, NOW(), 'user', 'individual'),
    (26, 'rebecca.nandutu@demo.test', 'a94a8fe5ccb19ba61c4c0873d391e987982fbbd3', 1026, 1001, NOW(), 'approver', 'facility'),
    (27, 'diana.nabulya@demo.test', 'a94a8fe5ccb19ba61c4c0873d391e987982fbbd3', 1027, 1026, NOW(), 'user', 'individual'),
    (28, 'noah.waiswa@demo.test', 'a94a8fe5ccb19ba61c4c0873d391e987982fbbd3', 1028, 1026, NOW(), 'user', 'individual'),
    (29, 'shamim.namusoke@demo.test', 'a94a8fe5ccb19ba61c4c0873d391e987982fbbd3', 1029, 1026, NOW(), 'user', 'individual'),
    (30, 'prossy.nabirye@demo.test', 'a94a8fe5ccb19ba61c4c0873d391e987982fbbd3', 1030, 1026, NOW(), 'user', 'individual'),
    (31, 'david.turyasingura@demo.test', 'a94a8fe5ccb19ba61c4c0873d391e987982fbbd3', 1031, 1001, NOW(), 'approver', 'facility'),
    (32, 'isaac.turyatemba@demo.test', 'a94a8fe5ccb19ba61c4c0873d391e987982fbbd3', 1032, 1031, NOW(), 'user', 'individual'),
    (33, 'sarah.kobusingye@demo.test', 'a94a8fe5ccb19ba61c4c0873d391e987982fbbd3', 1033, 1031, NOW(), 'user', 'individual'),
    (34, 'harriet.natukunda@demo.test', 'a94a8fe5ccb19ba61c4c0873d391e987982fbbd3', 1034, 1031, NOW(), 'user', 'individual'),
    (35, 'mercy.twinomujuni@demo.test', 'a94a8fe5ccb19ba61c4c0873d391e987982fbbd3', 1035, 1031, NOW(), 'user', 'individual'),
    (36, 'irene.namubiru@demo.test', 'a94a8fe5ccb19ba61c4c0873d391e987982fbbd3', 1036, 1001, NOW(), 'approver', 'facility'),
    (37, 'joan.nakirya@demo.test', 'a94a8fe5ccb19ba61c4c0873d391e987982fbbd3', 1037, 1036, NOW(), 'user', 'individual'),
    (38, 'richard.ssali@demo.test', 'a94a8fe5ccb19ba61c4c0873d391e987982fbbd3', 1038, 1036, NOW(), 'user', 'individual'),
    (39, 'moses.bbosa@demo.test', 'a94a8fe5ccb19ba61c4c0873d391e987982fbbd3', 1039, 1036, NOW(), 'user', 'individual'),
    (40, 'harold.sserwanja@demo.test', 'a94a8fe5ccb19ba61c4c0873d391e987982fbbd3', 1040, 1036, NOW(), 'user', 'individual'),
    (41, 'samuel.okello@demo.test', 'a94a8fe5ccb19ba61c4c0873d391e987982fbbd3', 1041, 1001, NOW(), 'approver', 'facility'),
    (42, 'charles.obong@demo.test', 'a94a8fe5ccb19ba61c4c0873d391e987982fbbd3', 1042, 1041, NOW(), 'user', 'individual'),
    (43, 'peninah.acan@demo.test', 'a94a8fe5ccb19ba61c4c0873d391e987982fbbd3', 1043, 1041, NOW(), 'user', 'individual'),
    (44, 'gloria.atim@demo.test', 'a94a8fe5ccb19ba61c4c0873d391e987982fbbd3', 1044, 1041, NOW(), 'user', 'individual'),
    (45, 'annet.acam@demo.test', 'a94a8fe5ccb19ba61c4c0873d391e987982fbbd3', 1045, 1041, NOW(), 'user', 'individual'),
    (46, 'agnes.nakato@demo.test', 'a94a8fe5ccb19ba61c4c0873d391e987982fbbd3', 1046, 1001, NOW(), 'approver', 'facility'),
    (47, 'esther.ssekimpi@demo.test', 'a94a8fe5ccb19ba61c4c0873d391e987982fbbd3', 1047, 1046, NOW(), 'user', 'individual'),
    (48, 'robert.kizito@demo.test', 'a94a8fe5ccb19ba61c4c0873d391e987982fbbd3', 1048, 1046, NOW(), 'user', 'individual'),
    (49, 'sharon.nakimuli@demo.test', 'a94a8fe5ccb19ba61c4c0873d391e987982fbbd3', 1049, 1046, NOW(), 'user', 'individual'),
    (50, 'rebecca.birungi@demo.test', 'a94a8fe5ccb19ba61c4c0873d391e987982fbbd3', 1050, 1046, NOW(), 'user', 'individual'),
    (51, 'peter.wekesa@demo.test', 'a94a8fe5ccb19ba61c4c0873d391e987982fbbd3', 1051, 1001, NOW(), 'approver', 'facility'),
    (52, 'ronald.masaba@demo.test', 'a94a8fe5ccb19ba61c4c0873d391e987982fbbd3', 1052, 1051, NOW(), 'user', 'individual'),
    (53, 'mildred.namatovu@demo.test', 'a94a8fe5ccb19ba61c4c0873d391e987982fbbd3', 1053, 1051, NOW(), 'user', 'individual'),
    (54, 'stephen.wanyama@demo.test', 'a94a8fe5ccb19ba61c4c0873d391e987982fbbd3', 1054, 1051, NOW(), 'user', 'individual'),
    (55, 'ivy.namuddu@demo.test', 'a94a8fe5ccb19ba61c4c0873d391e987982fbbd3', 1055, 1051, NOW(), 'user', 'individual'),
    (56, 'grace.tumusiime@demo.test', 'a94a8fe5ccb19ba61c4c0873d391e987982fbbd3', 1056, 1001, NOW(), 'approver', 'facility'),
    (57, 'patricia.ampaire@demo.test', 'a94a8fe5ccb19ba61c4c0873d391e987982fbbd3', 1057, 1056, NOW(), 'user', 'individual'),
    (58, 'brian.mugisha@demo.test', 'a94a8fe5ccb19ba61c4c0873d391e987982fbbd3', 1058, 1056, NOW(), 'user', 'individual'),
    (59, 'esther.busingye@demo.test', 'a94a8fe5ccb19ba61c4c0873d391e987982fbbd3', 1059, 1056, NOW(), 'user', 'individual'),
    (60, 'godfrey.taremwa@demo.test', 'a94a8fe5ccb19ba61c4c0873d391e987982fbbd3', 1060, 1056, NOW(), 'user', 'individual'),
    (61, 'simon.lokiru@demo.test', 'a94a8fe5ccb19ba61c4c0873d391e987982fbbd3', 1061, 1001, NOW(), 'approver', 'facility'),
    (62, 'emmanuel.longole@demo.test', 'a94a8fe5ccb19ba61c4c0873d391e987982fbbd3', 1062, 1061, NOW(), 'user', 'individual'),
    (63, 'doreen.aciro@demo.test', 'a94a8fe5ccb19ba61c4c0873d391e987982fbbd3', 1063, 1061, NOW(), 'user', 'individual'),
    (64, 'rose.akuj@demo.test', 'a94a8fe5ccb19ba61c4c0873d391e987982fbbd3', 1064, 1061, NOW(), 'user', 'individual'),
    (65, 'hilda.loputia@demo.test', 'a94a8fe5ccb19ba61c4c0873d391e987982fbbd3', 1065, 1061, NOW(), 'user', 'individual'),
    (66, 'ruth.namayanja@demo.test', 'a94a8fe5ccb19ba61c4c0873d391e987982fbbd3', 1066, 1001, NOW(), 'approver', 'facility'),
    (67, 'hellen.namusoke@demo.test', 'a94a8fe5ccb19ba61c4c0873d391e987982fbbd3', 1067, 1066, NOW(), 'user', 'individual'),
    (68, 'simon.kateregga@demo.test', 'a94a8fe5ccb19ba61c4c0873d391e987982fbbd3', 1068, 1066, NOW(), 'user', 'individual'),
    (69, 'lydia.ssenyonga@demo.test', 'a94a8fe5ccb19ba61c4c0873d391e987982fbbd3', 1069, 1066, NOW(), 'user', 'individual'),
    (70, 'brian.kasule@demo.test', 'a94a8fe5ccb19ba61c4c0873d391e987982fbbd3', 1070, 1066, NOW(), 'user', 'individual'),
    (71, 'joseph.ecom@demo.test', 'a94a8fe5ccb19ba61c4c0873d391e987982fbbd3', 1071, 1001, NOW(), 'approver', 'facility'),
    (72, 'vincent.ekuwam@demo.test', 'a94a8fe5ccb19ba61c4c0873d391e987982fbbd3', 1072, 1071, NOW(), 'user', 'individual'),
    (73, 'janet.elotu@demo.test', 'a94a8fe5ccb19ba61c4c0873d391e987982fbbd3', 1073, 1071, NOW(), 'user', 'individual'),
    (74, 'kenneth.omolo@demo.test', 'a94a8fe5ccb19ba61c4c0873d391e987982fbbd3', 1074, 1071, NOW(), 'user', 'individual'),
    (75, 'janet.apio@demo.test', 'a94a8fe5ccb19ba61c4c0873d391e987982fbbd3', 1075, 1071, NOW(), 'user', 'individual'),
    (76, 'miriam.nambuya@demo.test', 'a94a8fe5ccb19ba61c4c0873d391e987982fbbd3', 1076, 1001, NOW(), 'approver', 'facility'),
    (77, 'florah.nandera@demo.test', 'a94a8fe5ccb19ba61c4c0873d391e987982fbbd3', 1077, 1076, NOW(), 'user', 'individual'),
    (78, 'david.wandera@demo.test', 'a94a8fe5ccb19ba61c4c0873d391e987982fbbd3', 1078, 1076, NOW(), 'user', 'individual'),
    (79, 'naomi.nabwire@demo.test', 'a94a8fe5ccb19ba61c4c0873d391e987982fbbd3', 1079, 1076, NOW(), 'user', 'individual'),
    (80, 'claire.namutebi@demo.test', 'a94a8fe5ccb19ba61c4c0873d391e987982fbbd3', 1080, 1076, NOW(), 'user', 'individual'),
    (81, 'admin', 'd033e22ae348aeb5660fc2140aec35850c4da997', 1001, NULL, NOW(), 'admin', 'national'),
    (82, 'admin-region', 'a94a8fe5ccb19ba61c4c0873d391e987982fbbd3', 1006, 1001, NOW(), 'approver', 'facility');

INSERT INTO clinician_app.indicators (id, indicator, created_by, created_on) VALUES
    (1, 'Weekly specialist output', 1001, NOW()),
    (2, 'Staff attendance rate', 1001, NOW()),
    (3, 'Ward round coverage', 1001, NOW()),
    (4, 'Maternal and newborn service output', 1001, NOW()),
    (5, 'Diagnostic turnaround', 1001, NOW());

INSERT INTO clinician_app.targets (id, indicator, target, created_by, created_on) VALUES
    (1, 1, 40, 1001, NOW()),
    (2, 2, 95, 1001, NOW()),
    (3, 3, 25, 1001, NOW()),
    (4, 4, 18, 1001, NOW()),
    (5, 5, 24, 1001, NOW());

INSERT INTO clinician_app.department_roles (role_id, dept_id, role_name, data_points) VALUES
    (1, 1, 'default', '["attendance","ward_rounds","patients_reviewed","theatre_days","elective","emergency","postmortems","OPD_clinics","OPD_patients","teaching_rounds","students_taught","mortality_reviews","labs_requests","imaging_requests","investigations","xrays","ct_scans"]'::jsonb),
    (2, 2, 'default', '["attendance","ward_rounds","patients_reviewed","OPD_clinics","OPD_patients","teaching_rounds","students_taught","mortality_reviews","medical","labs_requests","imaging_requests","investigations","CBC","chemistry","hematology","urinalysis"]'::jsonb),
    (3, 3, 'default', '["attendance","ward_rounds","patients_reviewed","OPD_clinics","OPD_patients","teaching_rounds","students_taught","mortality_reviews","paed","labs_requests","imaging_requests","investigations","malaria","TB","CBC"]'::jsonb),
    (4, 4, 'default', '["attendance","ward_rounds","patients_reviewed","theatre_days","elective","emergency","anc_patients","maternal","perinatal","OPD_clinics","OPD_patients","teaching_rounds","students_taught","obstetrics_scans","abdominal_scans"]'::jsonb);

INSERT INTO clinician_app.staffleave (
    leave_id, employee_id, start_date, end_date, leave_status, approved_by, created_on, notes, leave_type_id, return_date
) VALUES
    (1, 1003, DATE '2026-02-01', DATE '2026-02-07', 'Valid', 1001, NOW(), 'Annual leave after duty roster completion', 1, DATE '2026-02-08'),
    (2, 1009, DATE '2026-02-05', DATE '2026-02-11', 'Completed', 1006, NOW(), 'Sick leave following acute febrile illness', 2, DATE '2026-02-12'),
    (3, 1015, DATE '2026-02-09', DATE '2026-02-15', 'Completed', 1011, NOW(), 'Maternity leave approved by hospital administration', 3, DATE '2026-02-16'),
    (4, 1018, DATE '2026-02-13', DATE '2026-02-19', 'Completed', 1016, NOW(), 'Study leave for specialist CPD update', 7, DATE '2026-02-20'),
    (5, 1024, DATE '2026-02-17', DATE '2026-02-23', 'Completed', 1021, NOW(), 'Annual leave after duty roster completion', 1, DATE '2026-02-24'),
    (6, 1030, DATE '2026-02-21', DATE '2026-02-27', 'Valid', 1026, NOW(), 'Sick leave following acute febrile illness', 2, DATE '2026-02-28'),
    (7, 1033, DATE '2026-02-25', DATE '2026-03-03', 'Completed', 1031, NOW(), 'Maternity leave approved by hospital administration', 3, DATE '2026-03-04'),
    (8, 1039, DATE '2026-03-01', DATE '2026-03-07', 'Completed', 1036, NOW(), 'Study leave for specialist CPD update', 7, DATE '2026-03-08'),
    (9, 1045, DATE '2026-03-05', DATE '2026-03-11', 'Completed', 1041, NOW(), 'Annual leave after duty roster completion', 1, DATE '2026-03-12'),
    (10, 1048, DATE '2026-03-09', DATE '2026-03-15', 'Completed', 1046, NOW(), 'Sick leave following acute febrile illness', 2, DATE '2026-03-16'),
    (11, 1054, DATE '2026-03-13', DATE '2026-03-19', 'Valid', 1051, NOW(), 'Maternity leave approved by hospital administration', 3, DATE '2026-03-20'),
    (12, 1060, DATE '2026-03-17', DATE '2026-03-23', 'Completed', 1056, NOW(), 'Study leave for specialist CPD update', 7, DATE '2026-03-24'),
    (13, 1063, DATE '2026-03-21', DATE '2026-03-27', 'Completed', 1061, NOW(), 'Annual leave after duty roster completion', 1, DATE '2026-03-28'),
    (14, 1069, DATE '2026-03-25', DATE '2026-03-31', 'Completed', 1066, NOW(), 'Sick leave following acute febrile illness', 2, DATE '2026-04-01'),
    (15, 1075, DATE '2026-03-29', DATE '2026-04-04', 'Completed', 1071, NOW(), 'Maternity leave approved by hospital administration', 3, DATE '2026-04-05'),
    (16, 1078, DATE '2026-04-02', DATE '2026-04-08', 'Valid', 1076, NOW(), 'Study leave for specialist CPD update', 7, DATE '2026-04-09');

INSERT INTO clinician_app.weeklyreport (
    id, hospital, department, employee, start, stop,
    qn_01, qn_02, qn_03, qn_04, qn_05, qn_06, qn_08, qn_09,
    qn_10, qn_11, qn_12, qn_13, qn_14, qn_15,
    qn_19, qn_20, qn_21, qn_22, qn_23, qn_24, qn_25, qn_26, qn_27, qn_28, qn_29,
    qn_35, qn_36, qn_37, qn_38,
    created_on, entered_by, report_status, last_updated_on, submitted_by, approved_by, submit_status
) VALUES
    (1, 1, 1, 1002, DATE '2026-04-13', DATE '2026-04-19', 5, 4, 28, 2, 6, 3, 2, 34, 0, 1, 18, 1, 0, 0, 12, 8, 9, 0, 0, 0, 0, 5, 6, 2, 3, 14, 4, 0, 0, NOW(), 1002, 'Draft', NOW(), NULL, NULL, NULL),
    (2, 2, 2, 1008, DATE '2026-04-13', DATE '2026-04-19', 5, 5, 43, 0, 0, 0, 4, 54, 0, 3, 22, 2, 0, 0, 15, 6, 12, 0, 0, 2, 1, 7, 8, 6, 4, 0, 0, 0, 0, NOW(), 1008, 'Approved', NOW(), 1008, 1006, 'Submitted'),
    (3, 3, 3, 1014, DATE '2026-04-13', DATE '2026-04-19', 5, 3, 42, 0, 0, 0, 5, 63, 0, 2, 24, 1, 0, 0, 18, 7, 15, 0, 0, 11, 4, 9, 3, 1, 8, 0, 0, 0, 0, NOW(), 1014, 'Approved', NOW(), 1014, 1011, 'Submitted'),
    (4, 4, 4, 1020, DATE '2026-04-13', DATE '2026-04-19', 4, 3, 22, 2, 4, 1, 3, 32, 18, 1, 14, 1, 5, 7, 7, 5, 4, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 10, 11, NOW(), 1020, 'Submitted', NOW(), 1020, NULL, 'Submitted'),
    (5, 5, 1, 1022, DATE '2026-04-13', DATE '2026-04-19', 5, 4, 29, 2, 6, 3, 2, 34, 0, 1, 18, 1, 0, 0, 12, 8, 9, 0, 0, 0, 0, 5, 6, 2, 3, 14, 4, 0, 0, NOW(), 1022, 'Approved', NOW(), 1022, 1021, 'Submitted'),
    (6, 6, 2, 1028, DATE '2026-04-13', DATE '2026-04-19', 5, 5, 44, 0, 0, 0, 4, 54, 0, 3, 22, 2, 0, 0, 15, 6, 12, 0, 0, 2, 1, 7, 8, 6, 4, 0, 0, 0, 0, NOW(), 1028, 'Draft', NOW(), NULL, NULL, NULL),
    (7, 7, 3, 1034, DATE '2026-04-13', DATE '2026-04-19', 5, 3, 40, 0, 0, 0, 5, 63, 0, 2, 24, 1, 0, 0, 18, 7, 15, 0, 0, 11, 4, 9, 3, 1, 8, 0, 0, 0, 0, NOW(), 1034, 'Submitted', NOW(), 1034, NULL, 'Submitted'),
    (8, 8, 4, 1040, DATE '2026-04-13', DATE '2026-04-19', 4, 3, 23, 2, 4, 1, 3, 32, 18, 1, 14, 1, 5, 7, 7, 5, 4, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 10, 11, NOW(), 1040, 'Approved', NOW(), 1040, 1036, 'Submitted'),
    (9, 9, 1, 1042, DATE '2026-04-13', DATE '2026-04-19', 5, 4, 30, 2, 6, 3, 2, 34, 0, 1, 18, 1, 0, 0, 12, 8, 9, 0, 0, 0, 0, 5, 6, 2, 3, 14, 4, 0, 0, NOW(), 1042, 'Approved', NOW(), 1042, 1041, 'Submitted'),
    (10, 10, 2, 1048, DATE '2026-04-13', DATE '2026-04-19', 5, 5, 42, 0, 0, 0, 4, 54, 0, 3, 22, 2, 0, 0, 15, 6, 12, 0, 0, 2, 1, 7, 8, 6, 4, 0, 0, 0, 0, NOW(), 1048, 'Submitted', NOW(), 1048, NULL, 'Submitted'),
    (11, 11, 3, 1054, DATE '2026-04-13', DATE '2026-04-19', 5, 3, 41, 0, 0, 0, 5, 63, 0, 2, 24, 1, 0, 0, 18, 7, 15, 0, 0, 11, 4, 9, 3, 1, 8, 0, 0, 0, 0, NOW(), 1054, 'Draft', NOW(), NULL, NULL, NULL),
    (12, 12, 4, 1060, DATE '2026-04-13', DATE '2026-04-19', 4, 3, 24, 2, 4, 1, 3, 32, 18, 1, 14, 1, 5, 7, 7, 5, 4, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 10, 11, NOW(), 1060, 'Approved', NOW(), 1060, 1056, 'Submitted'),
    (13, 13, 1, 1062, DATE '2026-04-13', DATE '2026-04-19', 5, 4, 28, 2, 6, 3, 2, 34, 0, 1, 18, 1, 0, 0, 12, 8, 9, 0, 0, 0, 0, 5, 6, 2, 3, 14, 4, 0, 0, NOW(), 1062, 'Submitted', NOW(), 1062, NULL, 'Submitted'),
    (14, 14, 2, 1068, DATE '2026-04-13', DATE '2026-04-19', 5, 5, 43, 0, 0, 0, 4, 54, 0, 3, 22, 2, 0, 0, 15, 6, 12, 0, 0, 2, 1, 7, 8, 6, 4, 0, 0, 0, 0, NOW(), 1068, 'Approved', NOW(), 1068, 1066, 'Submitted'),
    (15, 15, 3, 1074, DATE '2026-04-13', DATE '2026-04-19', 5, 3, 42, 0, 0, 0, 5, 63, 0, 2, 24, 1, 0, 0, 18, 7, 15, 0, 0, 11, 4, 9, 3, 1, 8, 0, 0, 0, 0, NOW(), 1074, 'Approved', NOW(), 1074, 1071, 'Submitted'),
    (16, 16, 4, 1080, DATE '2026-04-13', DATE '2026-04-19', 4, 3, 22, 2, 4, 1, 3, 32, 18, 1, 14, 1, 5, 7, 7, 5, 4, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 10, 11, NOW(), 1080, 'Draft', NOW(), NULL, NULL, NULL);

INSERT INTO clinician_app.attendance_records (
    id, attendance_date, specialist_id, department_id, attendance_type, facility_id
) VALUES
    (1, DATE '2026-04-14', 1002, 1, 'Present', 1),
    (2, DATE '2026-04-15', 1003, 2, 'Present', 1),
    (3, DATE '2026-04-14', 1008, 2, 'Present', 2),
    (4, DATE '2026-04-15', 1008, 2, 'Present', 2),
    (5, DATE '2026-04-14', 1014, 3, 'Present', 3),
    (6, DATE '2026-04-15', 1013, 2, 'Present', 3),
    (7, DATE '2026-04-14', 1020, 4, 'Present', 4),
    (8, DATE '2026-04-15', 1018, 2, 'Present', 4),
    (9, DATE '2026-04-14', 1022, 1, 'Present', 5),
    (10, DATE '2026-04-15', 1023, 2, 'Present', 5),
    (11, DATE '2026-04-14', 1028, 2, 'Present', 6),
    (12, DATE '2026-04-15', 1028, 2, 'Present', 6),
    (13, DATE '2026-04-14', 1034, 3, 'Present', 7),
    (14, DATE '2026-04-15', 1033, 2, 'Present', 7),
    (15, DATE '2026-04-14', 1040, 4, 'Present', 8),
    (16, DATE '2026-04-15', 1038, 2, 'Present', 8),
    (17, DATE '2026-04-14', 1042, 1, 'Present', 9),
    (18, DATE '2026-04-15', 1043, 2, 'Present', 9),
    (19, DATE '2026-04-14', 1048, 2, 'Present', 10),
    (20, DATE '2026-04-15', 1048, 2, 'Present', 10),
    (21, DATE '2026-04-14', 1054, 3, 'Present', 11),
    (22, DATE '2026-04-15', 1053, 2, 'Present', 11),
    (23, DATE '2026-04-14', 1060, 4, 'Present', 12),
    (24, DATE '2026-04-15', 1058, 2, 'Present', 12),
    (25, DATE '2026-04-14', 1062, 1, 'Present', 13),
    (26, DATE '2026-04-15', 1063, 2, 'Present', 13),
    (27, DATE '2026-04-14', 1068, 2, 'Present', 14),
    (28, DATE '2026-04-15', 1068, 2, 'Present', 14),
    (29, DATE '2026-04-14', 1074, 3, 'Present', 15),
    (30, DATE '2026-04-15', 1073, 2, 'Present', 15),
    (31, DATE '2026-04-14', 1080, 4, 'Present', 16),
    (32, DATE '2026-04-15', 1078, 2, 'Present', 16);

INSERT INTO clinician_app.surgeries (
    id, surgery_date, surgery_type, department_id, patient_id, surgeries_count, specialist_id, facility_id
) VALUES
    (1, DATE '2026-04-16', 'Elective Hernia Repair', 1, 7001, 1, 1002, 1),
    (2, DATE '2026-04-16', 'Emergency Laparotomy', 1, 7002, 2, 1007, 2),
    (3, DATE '2026-04-16', 'Caesarean Section', 4, 7003, 3, 1015, 3),
    (4, DATE '2026-04-16', 'Appendicectomy', 1, 7004, 1, 1017, 4),
    (5, DATE '2026-04-16', 'Elective Hernia Repair', 1, 7005, 2, 1022, 5),
    (6, DATE '2026-04-16', 'Emergency Laparotomy', 1, 7006, 3, 1027, 6),
    (7, DATE '2026-04-16', 'Caesarean Section', 4, 7007, 1, 1035, 7),
    (8, DATE '2026-04-16', 'Appendicectomy', 1, 7008, 2, 1037, 8),
    (9, DATE '2026-04-16', 'Elective Hernia Repair', 1, 7009, 3, 1042, 9),
    (10, DATE '2026-04-16', 'Emergency Laparotomy', 1, 7010, 1, 1047, 10),
    (11, DATE '2026-04-16', 'Caesarean Section', 4, 7011, 2, 1055, 11),
    (12, DATE '2026-04-16', 'Appendicectomy', 1, 7012, 3, 1057, 12),
    (13, DATE '2026-04-16', 'Elective Hernia Repair', 1, 7013, 1, 1062, 13),
    (14, DATE '2026-04-16', 'Emergency Laparotomy', 1, 7014, 2, 1067, 14),
    (15, DATE '2026-04-16', 'Caesarean Section', 4, 7015, 3, 1075, 15),
    (16, DATE '2026-04-16', 'Appendicectomy', 1, 7016, 1, 1077, 16);

INSERT INTO clinician_app.ward_rounds (
    id, round_date, department_id, patients_reviewed, specialist_id, facility_id
) VALUES
    (1, DATE '2026-04-14', 1, 8, 1002, 1),
    (2, DATE '2026-04-14', 2, 9, 1008, 2),
    (3, DATE '2026-04-14', 3, 10, 1014, 3),
    (4, DATE '2026-04-14', 4, 11, 1020, 4),
    (5, DATE '2026-04-14', 1, 12, 1022, 5),
    (6, DATE '2026-04-14', 2, 13, 1028, 6),
    (7, DATE '2026-04-14', 3, 14, 1034, 7),
    (8, DATE '2026-04-14', 4, 15, 1040, 8),
    (9, DATE '2026-04-14', 1, 16, 1042, 9),
    (10, DATE '2026-04-14', 2, 8, 1048, 10),
    (11, DATE '2026-04-14', 3, 9, 1054, 11),
    (12, DATE '2026-04-14', 4, 10, 1060, 12),
    (13, DATE '2026-04-14', 1, 11, 1062, 13),
    (14, DATE '2026-04-14', 2, 12, 1068, 14),
    (15, DATE '2026-04-14', 3, 13, 1074, 15),
    (16, DATE '2026-04-14', 4, 14, 1080, 16);

INSERT INTO clinician_app.investigations (
    id, request_date, investigation_type, test_type, result_status, result, specialist_id, facility_id
) VALUES
    (1, DATE '2026-04-16', 'Laboratory', 'CBC', 'Completed', 'Mild anaemia', 1002, 1),
    (2, DATE '2026-04-16', 'Imaging', 'Chest X-Ray', 'Completed', 'No acute cardiopulmonary abnormality', 1008, 2),
    (3, DATE '2026-04-16', 'Laboratory', 'Malaria Rapid Test', 'Completed', 'Negative', 1014, 3),
    (4, DATE '2026-04-16', 'Imaging', 'Obstetric Ultrasound', 'Completed', 'Singleton live intrauterine pregnancy', 1020, 4),
    (5, DATE '2026-04-16', 'Laboratory', 'CBC', 'Completed', 'Mild anaemia', 1022, 5),
    (6, DATE '2026-04-16', 'Imaging', 'Chest X-Ray', 'Completed', 'No acute cardiopulmonary abnormality', 1028, 6),
    (7, DATE '2026-04-16', 'Laboratory', 'Malaria Rapid Test', 'Completed', 'Negative', 1034, 7),
    (8, DATE '2026-04-16', 'Imaging', 'Obstetric Ultrasound', 'Completed', 'Singleton live intrauterine pregnancy', 1040, 8),
    (9, DATE '2026-04-16', 'Laboratory', 'CBC', 'Completed', 'Mild anaemia', 1042, 9),
    (10, DATE '2026-04-16', 'Imaging', 'Chest X-Ray', 'Completed', 'No acute cardiopulmonary abnormality', 1048, 10),
    (11, DATE '2026-04-16', 'Laboratory', 'Malaria Rapid Test', 'Completed', 'Negative', 1054, 11),
    (12, DATE '2026-04-16', 'Imaging', 'Obstetric Ultrasound', 'Completed', 'Singleton live intrauterine pregnancy', 1060, 12),
    (13, DATE '2026-04-16', 'Laboratory', 'CBC', 'Completed', 'Mild anaemia', 1062, 13),
    (14, DATE '2026-04-16', 'Imaging', 'Chest X-Ray', 'Completed', 'No acute cardiopulmonary abnormality', 1068, 14),
    (15, DATE '2026-04-16', 'Laboratory', 'Malaria Rapid Test', 'Completed', 'Negative', 1074, 15),
    (16, DATE '2026-04-16', 'Imaging', 'Obstetric Ultrasound', 'Completed', 'Singleton live intrauterine pregnancy', 1080, 16);

SELECT setval(pg_get_serial_sequence('clinician_app.lg', 'id'), COALESCE(MAX(id), 1), true) FROM clinician_app.lg;
SELECT setval(pg_get_serial_sequence('clinician_app.facilities', 'id'), COALESCE(MAX(id), 1), true) FROM clinician_app.facilities;
SELECT setval(pg_get_serial_sequence('clinician_app.departments', 'id'), COALESCE(MAX(id), 1), true) FROM clinician_app.departments;
SELECT setval(pg_get_serial_sequence('clinician_app.rights', 'id'), COALESCE(MAX(id), 1), true) FROM clinician_app.rights;
SELECT setval(pg_get_serial_sequence('clinician_app.employeerights', 'id'), COALESCE(MAX(id), 1), true) FROM clinician_app.employeerights;
SELECT setval(pg_get_serial_sequence('clinician_app.users', 'id'), COALESCE(MAX(id), 1), true) FROM clinician_app.users;
SELECT setval(pg_get_serial_sequence('clinician_app.indicators', 'id'), COALESCE(MAX(id), 1), true) FROM clinician_app.indicators;
SELECT setval(pg_get_serial_sequence('clinician_app.targets', 'id'), COALESCE(MAX(id), 1), true) FROM clinician_app.targets;
SELECT setval(pg_get_serial_sequence('clinician_app.department_roles', 'role_id'), COALESCE(MAX(role_id), 1), true) FROM clinician_app.department_roles;
SELECT setval(pg_get_serial_sequence('clinician_app.attendance_records', 'id'), COALESCE(MAX(id), 1), true) FROM clinician_app.attendance_records;
SELECT setval(pg_get_serial_sequence('clinician_app.surgeries', 'id'), COALESCE(MAX(id), 1), true) FROM clinician_app.surgeries;
SELECT setval(pg_get_serial_sequence('clinician_app.ward_rounds', 'id'), COALESCE(MAX(id), 1), true) FROM clinician_app.ward_rounds;
SELECT setval(pg_get_serial_sequence('clinician_app.investigations', 'id'), COALESCE(MAX(id), 1), true) FROM clinician_app.investigations;
