*&---------------------------------------------------------------------*
*& Report ZOPEN_RFC_TRAFFIC_GEN
*&---------------------------------------------------------------------*
*& Generate varied classic-RFC traffic to a destination, so it can be
*& captured through a proxy (cmd/rfc-sniffer) and decoded by cmd/rfc-viewer.
*&
*& Run in the CALLING system (e.g. i105). P_DEST must be an SM59 type-3
*& (ABAP Connection) destination whose target host points at the sniffer
*& machine, same instance number as the real target (so the port matches),
*& and the sniffer forwards to the real target (e.g. i103).
*&
*& Each call is guarded with EXCEPTIONS so a missing/mismatched FM only sets
*& SY-SUBRC and never dumps. It exercises: empty ping, a structure export,
*& scalar in/out, structure+table, a table read, and (optionally) STRING and
*& deep/nested xRFC — the serializations most worth capturing.
*&---------------------------------------------------------------------*
REPORT zopen_rfc_traffic_gen.

PARAMETERS: p_dest  TYPE rfcdest DEFAULT 'I3_VIA_SNIFF' OBLIGATORY,
            p_deep  AS CHECKBOX DEFAULT 'X',   " also call STFC_DEEP_* / STFC_STRING
            p_read  AS CHECKBOX DEFAULT 'X'.   " also call RFC_READ_TABLE

DATA: lv_echo TYPE c LENGTH 255,
      lv_resp TYPE c LENGTH 255,
      lv_rc   TYPE sy-subrc.

START-OF-SELECTION.

  WRITE: / '== RFC traffic generator ->', p_dest, '=='.

*--- 1) RFC_PING: empty round trip (the SM59 Connection Test call) ------
  CALL FUNCTION 'RFC_PING' DESTINATION p_dest
    EXCEPTIONS communication_failure = 1 system_failure = 2 OTHERS = 3.
  WRITE: / 'RFC_PING              rc=', sy-subrc.

*--- 2) RFC_SYSTEM_INFO: structure export (the Unicode / fast-ser test) --
  DATA ls_info TYPE rfcsi.
  CALL FUNCTION 'RFC_SYSTEM_INFO' DESTINATION p_dest
    IMPORTING rfcsi_export = ls_info
    EXCEPTIONS communication_failure = 1 system_failure = 2 OTHERS = 3.
  WRITE: / 'RFC_SYSTEM_INFO       rc=', sy-subrc, 'sysid=', ls_info-rfcsysid.

*--- 3) STFC_CONNECTION: scalar in / scalar out ------------------------
  CALL FUNCTION 'STFC_CONNECTION' DESTINATION p_dest
    EXPORTING requtext = 'hello through the sniffer'
    IMPORTING echotext = lv_echo
              resptext = lv_resp
    EXCEPTIONS communication_failure = 1 system_failure = 2 OTHERS = 3.
  WRITE: / 'STFC_CONNECTION       rc=', sy-subrc, 'echo=', lv_echo(30).

*--- 4) STFC_STRUCTURE: structure import/export + a table --------------
  DATA: ls_imp  TYPE rfctest,
        ls_echo TYPE rfctest,
        lt_tab  TYPE STANDARD TABLE OF rfctest.
  ls_imp-rfcchar1 = 'A'.
  ls_imp-rfcdata1 = 'structure through the sniffer'.
  CALL FUNCTION 'STFC_STRUCTURE' DESTINATION p_dest
    EXPORTING importstruct = ls_imp
    IMPORTING echostruct   = ls_echo
              resptext     = lv_resp
    TABLES    rfctable     = lt_tab
    EXCEPTIONS communication_failure = 1 system_failure = 2 OTHERS = 3.
  WRITE: / 'STFC_STRUCTURE        rc=', sy-subrc, 'rows=', lines( lt_tab ).

*--- 5) RFC_READ_TABLE: read a few rows of a small dictionary table ----
  IF p_read = 'X'.
    DATA: lt_data    TYPE STANDARD TABLE OF tab512,
          lt_fields  TYPE STANDARD TABLE OF rfc_db_fld,
          lt_options TYPE STANDARD TABLE OF rfc_db_opt.
    CALL FUNCTION 'RFC_READ_TABLE' DESTINATION p_dest
      EXPORTING query_table = 'T000'
                rowcount    = 5
      TABLES    options     = lt_options
                fields      = lt_fields
                data        = lt_data
      EXCEPTIONS communication_failure = 1 system_failure = 2 OTHERS = 3.
    WRITE: / 'RFC_READ_TABLE        rc=', sy-subrc, 'cols=', lines( lt_fields ),
             'rows=', lines( lt_data ).
  ENDIF.

*--- 6) STRING + deep/nested xRFC (the xRFC serializations) ------------
  IF p_deep = 'X'.
    " STFC_STRING: STRING scalar -> xRFC XML on the wire
    DATA: lv_in  TYPE string,
          lv_out TYPE string.
    lv_in = 'xrfc string via the sniffer'.
    CALL FUNCTION 'STFC_STRING' DESTINATION p_dest
      EXPORTING echostr = lv_in
      IMPORTING importstr = lv_out
      EXCEPTIONS communication_failure = 1 system_failure = 2 OTHERS = 3.
    WRITE: / 'STFC_STRING           rc=', sy-subrc.

    " STFC_DEEP_STRUCTURE: nested/recursive structure -> recursive xRFC.
    " Parameter names/types vary by release; guarded so a mismatch only sets
    " SY-SUBRC. If it is not available, comment it out or adjust in SE37.
    " CALL FUNCTION 'STFC_DEEP_STRUCTURE' DESTINATION p_dest
    "   EXPORTING import_deep_structure = ls_deep
    "   IMPORTING export_deep_structure = ls_deep_out
    "   RESPTEXT  = lv_resp
    "   EXCEPTIONS communication_failure = 1 system_failure = 2 OTHERS = 3.
    " WRITE: / 'STFC_DEEP_STRUCTURE   rc=', sy-subrc.
  ENDIF.

  WRITE: / '== done — check the sniffer capture / rfc-viewer =='.
