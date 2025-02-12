PGDMP      *    	             }         	   clinician    17.0    17.0 *    �           0    0    ENCODING    ENCODING        SET client_encoding = 'UTF8';
                           false            �           0    0 
   STDSTRINGS 
   STDSTRINGS     (   SET standard_conforming_strings = 'on';
                           false            �           0    0 
   SEARCHPATH 
   SEARCHPATH     8   SELECT pg_catalog.set_config('search_path', '', false);
                           false            �           1262    16388 	   clinician    DATABASE     �   CREATE DATABASE clinician WITH TEMPLATE = template0 ENCODING = 'UTF8' LOCALE_PROVIDER = libc LOCALE = 'English_United States.1252';
    DROP DATABASE clinician;
                     postgres    false            �          0    16467    attendance_records 
   TABLE DATA           }   COPY public.attendance_records (id, attendance_date, specialist_id, department_id, attendance_type, facility_id) FROM stdin;
    public               postgres    false    238   '       �          0    24710    department_roles 
   TABLE DATA           T   COPY public.department_roles (role_id, dept_id, role_name, data_points) FROM stdin;
    public               postgres    false    254   0'       �          0    16549    department_roles-old 
   TABLE DATA           P   COPY public."department_roles-old" (id, role, data_points, dept_id) FROM stdin;
    public               postgres    false    246   �(       �          0    16389    departments 
   TABLE DATA           1   COPY public.departments (id, d_name) FROM stdin;
    public               postgres    false    217   �)       �          0    16393    employeerights 
   TABLE DATA           >   COPY public.employeerights (id, employee, rights) FROM stdin;
    public               postgres    false    219   �*       �          0    16397 	   employees 
   TABLE DATA           �   COPY public.employees (id, fname, lname, oname, specialisation, department, facility, created_by, created_on, title) FROM stdin;
    public               postgres    false    221   �*       �          0    16402 
   facilities 
   TABLE DATA           W   COPY public.facilities (id, f_name, f_level, f_lg, created_by, created_on) FROM stdin;
    public               postgres    false    223   �1       �          0    16407 
   indicators 
   TABLE DATA           K   COPY public.indicators (id, indicator, created_by, created_on) FROM stdin;
    public               postgres    false    225   3       �          0    16489    investigations 
   TABLE DATA           �   COPY public.investigations (id, request_date, investigation_type, test_type, result_status, result, specialist_id, facility_id) FROM stdin;
    public               postgres    false    244   $3       �          0    24686 
   leavetypes 
   TABLE DATA           i   COPY public.leavetypes (leave_type_id, leave_type_name, description, created_at, updated_at) FROM stdin;
    public               postgres    false    250   A3       �          0    16412    lg 
   TABLE DATA           2   COPY public.lg (id, lg_name, lg_type) FROM stdin;
    public               postgres    false    227   �4       �          0    16416    rights 
   TABLE DATA           ,   COPY public.rights (id, rights) FROM stdin;
    public               postgres    false    229   �4       �          0    24697    specialist_titles 
   TABLE DATA           6   COPY public.specialist_titles (id, title) FROM stdin;
    public               postgres    false    252    5       �          0    24671 
   staffleave 
   TABLE DATA           �   COPY public.staffleave (leave_id, employee_id, start_date, end_date, leave_status, approved_by, created_on, notes, leave_type_id, return_date) FROM stdin;
    public               postgres    false    248   �5       �          0    16481 	   surgeries 
   TABLE DATA           �   COPY public.surgeries (id, surgery_date, surgery_type, department_id, patient_id, surgeries_count, specialist_id, facility_id) FROM stdin;
    public               postgres    false    242   c6       �          0    16420    targets 
   TABLE DATA           P   COPY public.targets (id, indicator, target, created_by, created_on) FROM stdin;
    public               postgres    false    231   �6       �          0    16425    users 
   TABLE DATA           a   COPY public.users (id, username, pssword, employees, created_by, created_on, rights) FROM stdin;
    public               postgres    false    233   �6       �          0    16474    ward_rounds 
   TABLE DATA           s   COPY public.ward_rounds (id, round_date, department_id, patients_reviewed, specialist_id, facility_id) FROM stdin;
    public               postgres    false    240   �7       �          0    16430    weeklyreport 
   TABLE DATA           �  COPY public.weeklyreport (id, hospital, department, employee, start, stop, qn_01, qn_02, qn_03, qn_04, qn_05, qn_06, qn_07, qn_08, qn_09, qn_10, created_on, entered_by, report_status, qn_11, qn_12, qn_13, qn_14, qn_15, qn_16, qn_17, qn_18, qn_19, qn_20, qn_21, qn_22, qn_23, qn_24, qn_25, qn_26, qn_27, qn_28, qn_29, qn_30, qn_31, qn_32, qn_33, qn_34, qn_35, qn_36, qn_37, qn_38, qn_39, qn_40, last_updated_on, submitted_by, approved_by, submit_status) FROM stdin;
    public               postgres    false    235   �7       �           0    0    attendance_records_id_seq    SEQUENCE SET     H   SELECT pg_catalog.setval('public.attendance_records_id_seq', 1, false);
          public               postgres    false    237            �           0    0 "   department_role_data_points_id_seq    SEQUENCE SET     P   SELECT pg_catalog.setval('public.department_role_data_points_id_seq', 5, true);
          public               postgres    false    245            �           0    0    department_roles_role_id_seq    SEQUENCE SET     J   SELECT pg_catalog.setval('public.department_roles_role_id_seq', 8, true);
          public               postgres    false    253            �           0    0    departments_id_seq    SEQUENCE SET     @   SELECT pg_catalog.setval('public.departments_id_seq', 4, true);
          public               postgres    false    218            �           0    0    employeerights_id_seq    SEQUENCE SET     D   SELECT pg_catalog.setval('public.employeerights_id_seq', 1, false);
          public               postgres    false    220            �           0    0    employees_id_seq    SEQUENCE SET     >   SELECT pg_catalog.setval('public.employees_id_seq', 9, true);
          public               postgres    false    222            �           0    0    facilities_id_seq    SEQUENCE SET     ?   SELECT pg_catalog.setval('public.facilities_id_seq', 4, true);
          public               postgres    false    224            �           0    0    indicators_id_seq    SEQUENCE SET     @   SELECT pg_catalog.setval('public.indicators_id_seq', 1, false);
          public               postgres    false    226            �           0    0    investigations_id_seq    SEQUENCE SET     D   SELECT pg_catalog.setval('public.investigations_id_seq', 1, false);
          public               postgres    false    243            �           0    0    leavetypes_leave_type_id_seq    SEQUENCE SET     K   SELECT pg_catalog.setval('public.leavetypes_leave_type_id_seq', 10, true);
          public               postgres    false    249            �           0    0 	   lg_id_seq    SEQUENCE SET     7   SELECT pg_catalog.setval('public.lg_id_seq', 2, true);
          public               postgres    false    228            �           0    0    rights_id_seq    SEQUENCE SET     ;   SELECT pg_catalog.setval('public.rights_id_seq', 7, true);
          public               postgres    false    230            �           0    0    specialist_titles_id_seq    SEQUENCE SET     G   SELECT pg_catalog.setval('public.specialist_titles_id_seq', 1, false);
          public               postgres    false    251            �           0    0    staffleave_leave_id_seq    SEQUENCE SET     F   SELECT pg_catalog.setval('public.staffleave_leave_id_seq', 1, false);
          public               postgres    false    247            �           0    0    surgeries_id_seq    SEQUENCE SET     ?   SELECT pg_catalog.setval('public.surgeries_id_seq', 1, false);
          public               postgres    false    241            �           0    0    targets_id_seq    SEQUENCE SET     =   SELECT pg_catalog.setval('public.targets_id_seq', 1, false);
          public               postgres    false    232            �           0    0    users_id_seq    SEQUENCE SET     :   SELECT pg_catalog.setval('public.users_id_seq', 1, true);
          public               postgres    false    234            �           0    0    ward_rounds_id_seq    SEQUENCE SET     A   SELECT pg_catalog.setval('public.ward_rounds_id_seq', 1, false);
          public               postgres    false    239            �           0    0    weeklyreport_id_seq    SEQUENCE SET     A   SELECT pg_catalog.setval('public.weeklyreport_id_seq', 1, true);
          public               postgres    false    236            �      x������ � �      �   �  x��TMk�0=˿�칔�����Ihm7$!�6��4���J����h���Jo��f͛�gͼWg��I}[3z^��M��aJb�.1r$�ڤ��t&8�`�Cՙj���C��mne�tu/��@̻���s���I�lg(�&��ܔ���#�61�/i��1�2�t��%����1�P=Q'�x(��`qƖ�@�oc�)%�=��a���.$v!2:�]����1󬿠!M~����������	��������p���	1�r�=t���WJ�H�{'��'���p?����B'�E� }�p�=T�j��9�c����9[���Т����أ���+ё:R�_i�E��X��˴������Q�>���Su��1O����^$�K�A5ku��n�f������v7� �-]�������)      �   �   x�}��
�0Dϛo	�B?C�k!,f�u#ɪ����qv��N�9�;��x���ɻg����OLَ,1��>�15<�s�İ]�0t��㬰O1��Ҙ-�BY9�r�n��a��3^��AJ�N^�ʴ�Q�k��N���D�Z�ڼoƘ/��b^      �   (  x�M��n�0���S�	&R`���؆4
b;r1��D
	r�I}���nN����;6A��bK�3.�*a��FagR��.V����$�b��[n�;5��)	�u] }l:5�&b��3�8ח�ըط'���o�ݠL6gl\h��(z�'���E��i����S	+�3�r
���gP���=���;N����ȫ3��$���8�!	�F���z[��N�5��]8��u��a��_bq�I��	��$��á��ҧ(r��]�DC��rDQe	_x��3���:>)���O�;      �      x������ � �      �   �  x��W�RI|��
=͛�u�~3���c'b�/%�V*�����2��7O0��"�8�sɓyJYv����W�}�x�����:�{&��1ɥ~#�Q�DQq^	�+c,Ld�`���P�/C��]l�C�&5L1��3�*n*-�Rsn���]�m�.�nX���gV��m\vH�0��"�L�� �"��sN2�)��V��]��}z�P���(+U�©R
@h�>�M���������E7�mv�o���w��9/
'�x�����mس��k�p��ߤ:���x"���̭��
f2���o��[&v��m�h�:������u����>�]����߲�q�k�Q<F�7�RRU��B��a�����F�>Ŧ���o�]\���_�C�T�fj-W��0��������0���2��~���@+��}�۱����v�z9㯔έ�FKD�R�ٳ�o���oL�!���͋Rh0Gd��ji`W���~�����Lp�%5h�<���x�]%Zg��i\�в�6�?����B�-
�Q\��=���w��W��yiP�*DA�����=����ݳ5�@0�,r��54��ۥ�]oC��������=@�)K+��,�9w��m����i�0*�[�bXh��(��L8��sW����s��m������L��2�}����zH�?݁W0Ԓ�M\v��|���Lēv5����w�f3��i��=���y�Pz৚���<4�˲p���`���w�Y��LL� zmJD���xh��V���E	���𫘀�_��R�:Vf���X{3l��pw/��q/Sp�Q�9���:��:�!��GO/@�R3��8R�/��ZX�+]AJ��墻�@�aQ���C�u�aŴ�bBR:�	2k�Y��"��/�����$鑂K9*�e���uD�l!���O�������`�V~�Na�ull�����ʍ%�;�d3Iޱ���is� N�gw���.�GtX)�2iؿb�5�e���쁣~����@��%��P�q��Z�%+ u٧ߏ6H�� \%6X é��"BKF[o~mB����֒)J������-~��t�R��di��5/D��U�8�-���m�L�%��k�5�YBA�~���;���9l��n��
�#�ϷQ�ϕ�K�hL�Lg����chz��c�������H���KF�N����258YN2 �OBW��b�X��;��|$g'�n�=C]���L��+R	Ì`�h��e׻t�<&��Pm�"NS�P�W:�K�9�a[,-C�[�I]����+�
�)�k�-���ݷ������'�jBG���_��Q�S����#C&Ȫ�ʕ�]�iA����?��,��,%���/	/�ڑ��������⨬[����A_��*�U�����|C��`+�5�<��B�$5�h�6syc�d���ؕ!2�h$���o�a���`KW���Z�9c	�������ē�����[rx\o�^�Ԧ��|�zt�����׻Ce8Q'��A0I[�^4M����l���3�L��Pn<�J*
(�36x�ػ=� ����$r�9$C(�x8�u9�S'� �-s-B��g���=���S��~K�J����$��K<_����p�G��r�Ƿ��d�tx�|Bx�'����L���%zi
I���<˲�U��      �   -  x���AK�0��˧��SK^�dmo*n��6D�]RF\ی�Q����M�G��ǟ�#n�hZw2��ր�\�yʋU�ˊ�Ԝsdk�n����e�y%y�d�e�J3M���'��}q�s{H�|o���_�E%t��B��v!"%���9����eM	��Q��C4��_p7��m�~���>��階�C�]O���Z�Y��mn���Kh�4>�ѓ������ю��9s0�:ʩ�8IZp�w�+�"�bIT�s�[R|j���a���*嘢��TX)��\`��>c�}i�ߨ      �      x������ � �      �      x������ � �      �   4  x����j�@��7O1/���j�][
�hA���fܝ���cZ���h)j!�fa��|3gN��I�����ᅈ{2����8�;�N�^�����ydլ�O�rR.��Wu���Ӻn�Es�TTl#����h"	)�`A*e(�4	��0XF"k������?�mү�%�l���9[�	s��c�����\�t�� ��Z�2X�ޒ�^�w�P�3	����)�=��͗l��� ��1Jkr8�7Riv��yϞ�v��9�E<?QݖΆ��VI/VI�;����;
Ƃ�{��wd�cr��Ӣ(���4�      �   .   x�3��N�-H�I�t�,��2�O��,��t�,.)�L.����� �3H      �   M   x�-�9� ����D���4�0��C�����͜d�1�"]���7�C����\��%W���AA�ϼ+�x�      �   m   x�3��MM�LN�Q�OK�LN-�v��2B�2�TW������Y\��e�����_����W\�S��W�eʉ�1�)�+-*��K��kΉ.b�阒��4�(�$3?�+F��� J7�      �   �   x�u���0���S�A��8?ѥR'��]P��s��@+�N����B����W�����=>Q�'N�c%�%����"8,S�I6i�v�S�R�zU-� i���񌡕m�a�'�1��8fH�Ti�PX�0��?}�1�>�*�>r	2���5�	�$�&�I�\Z������z����O�      �      x������ � �      �      x������ � �      �     x���=n�0Fg������4�=A92\!�]8���U��@6�|��#C�)χLǾL񼔘Wy��Am]dL�yJ)���E��H��S�L3^##�p`:�	�*�)oA��XWY�x�j�p�x	e�k�}�e�I�-�꟨h�q�:�ʀʣ?񞷥]���/y�A��[z�|�A�|X��ó�HB�jG(�n�ط-�=��r0FZ��v�b���G�r=�=�'�۠Tތηt�R?k������1&�F�~nv.R�c��      �      x������ � �      �   �  x��ZK��6]S��
�Ç�\��lg��+�EW���� G$ �5�ѐKF�D�_��p� ۡE~�l�8�|1����kO��p`�#w�������?��n�n�>^��K!��1>���{&d!���|9a��0�s��-���q�k�q|����L��_�uH��ǎ2A��Uc�f��^���EϏ��������_��}=��(Գ.�P'���1��`�t�<Hf�3�4��?/��B�H��u��~������}�XI��J.F�9؆������V���V���9H����7��^����` �u�m�\
�;- ��,��K���q�O���ǯ�)Ѵ�	F� �&�y��+�,��!��V�e�ज़�|��J��l�YTφY}bT2a%��DC�6]�=F�6Z���؍ݽTI��%���\����L6�3�Ʊ��&����|�#���u�^�
qp"��2{N~X��W�f)]l�F� �t��d���_���.�?�Di�kH�3!+~_Br =�	�}@u��HIw;�c�ǐgi�L�>B�	ԏ@y�QT��RX����:�+.%�#$��M&�DL���IZhں�5E%쵈����	AVt��n�6^#V�ʠ�5��ɗZ�Zj�G��z�_�i���;e�p��Tv�q
�0%�Gc��\G��U���
�4�&���Յ2DrO4�%&�K �� �'�}����A 4M�A�g�}��<�8B�hd㉜F����ߗX��#3f�{B�F(�#�B�N��Ż�#�޺o�r���e��(cF�Ma��GC�Mdca� r�p��᳂:��;
��v���u�J��
��U�z�X���y��ܣ�ʄO�����RoG��:_�9q%���K�JR�e����|	C�J�M2�ɕjKLy���m��P�Abن������b|KQw�2کԩq0 D����|Nd��P�x��P�S[q1�WC�
�'M9��ȉ�}c�{�&���M�|���i�+��n�Q���J��(���N���j���%���j����Ph�;M!�â�}8�ǯs��B/bH@��2���Z�6߅^���a��i��}H��yG�_�z'������cוs{7,YW�h]��b�R �e�&�1���_
�4�K�O�sdeV�>h�}%�OM.���������!5���}���i0 <����)���������9�AU�i�4�\~A&=k�e�c,�_�A�R�����L僅�4i�d�����WKiǸ0&�����H���?�[�|����ckȡg�db�F�` � �Bz�-��@(�`�C��� ���p�ba�Ք�1�B�r�K
���-c!\�⛹��jh��[' u!���En�V��߰�WNjM�WAe_��i���Qd�w�Nֺ.�`>!!�+�i��i����לd8��k��u�`A=6�;���!��IXem��n;ʨ)��cM$����+}���m&�*E����}��<.� �J���-�$)���q��%�K{4U���}O�&�NW�T�`�`�6�7Z�G�J�Y�CHZB�0B(���p��FA#���U��AM�k�~�LqYѷ �ԩ��Dp6L��S(ކLt/��ok>)?S��:	��Yi>�]ͧ�:���|沦ĢkpL �x�� �m�m ���p/�X���7��UοW!�l{ߟ�>�4�������R5��A0j�z]��q�;�D}1���|^�L     