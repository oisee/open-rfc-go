*&---------------------------------------------------------------------*
*& Report ZOPEN_RFC_BRIDGE_TEST
*&---------------------------------------------------------------------*
*& Exercises the open-rfc-go "conscious" server / polyglot bridge: ABAP
*& calls function modules whose logic runs in Go (cmd/rfc-lab, the
*& conscious endpoint), which GENERATES the RFC responses — no capture
*& replay.
*&
*& SETUP
*&  1) Run the lab:  go run ./cmd/rfc-lab -target-host <real-sap-host>
*&  2) SM59 destination P_DEST (type 3, ABAP Connection):
*&       Technical Settings: Target Host = <lab host>, System Number = 13
*&         (the conscious server listens on 33NN = 3313)
*&       Special Options -> Select Transfer Protocol -> Serializer =
*&         **Classic serializer**   (the server emits classic serialization)
*&  3) The two custom FMs must exist in THIS system as Remote-Enabled
*&     modules (SE37). Only the interface matters — the body never runs
*&     here; it executes on the Go server. Create:
*&
*&     Z_DOUBLE  (Remote-Enabled Module)
*&        IMPORTING  VALUE(N)      TYPE I
*&        EXPORTING  VALUE(RESULT) TYPE I
*&
*&     Z_GREET   (Remote-Enabled Module)
*&        IMPORTING  VALUE(NAME)     TYPE CHAR30
*&        EXPORTING  VALUE(GREETING) TYPE CHAR90
*&
*& STFC_CONNECTION already exists everywhere, so it needs no stub.
*&---------------------------------------------------------------------*
REPORT zopen_rfc_bridge_test.

PARAMETERS: p_dest TYPE rfcdest DEFAULT 'A4H@GEN' OBLIGATORY.

START-OF-SELECTION.

  WRITE: / '== open-rfc-go polyglot bridge test ->', p_dest, '=='.

*--- 1) STFC_CONNECTION: echo, answered by Go ------------------------
  DATA: lv_echo TYPE c LENGTH 255,
        lv_resp TYPE c LENGTH 255.
  CALL FUNCTION 'STFC_CONNECTION' DESTINATION p_dest
    EXPORTING requtext = 'ping from ABAP'
    IMPORTING echotext = lv_echo
              resptext = lv_resp
    EXCEPTIONS communication_failure = 1 system_failure = 2 OTHERS = 3.
  WRITE: / 'STFC_CONNECTION  rc=', sy-subrc.
  WRITE: / '   ECHOTEXT =', lv_echo(40).
  WRITE: / '   RESPTEXT =', lv_resp(40).

*--- 2) Z_DOUBLE: an ordinary Go function, called as an FM -----------
  DATA lv_result TYPE i.
  CALL FUNCTION 'Z_DOUBLE' DESTINATION p_dest
    EXPORTING n      = 21
    IMPORTING result = lv_result
    EXCEPTIONS communication_failure = 1 system_failure = 2 OTHERS = 3.
  WRITE: / 'Z_DOUBLE(21)     rc=', sy-subrc, ' RESULT =', lv_result.

*--- 3) Z_GREET: Go string logic over RFC ---------------------------
  DATA lv_greeting TYPE c LENGTH 90.
  CALL FUNCTION 'Z_GREET' DESTINATION p_dest
    EXPORTING name     = 'world'
    IMPORTING greeting = lv_greeting
    EXCEPTIONS communication_failure = 1 system_failure = 2 OTHERS = 3.
  WRITE: / 'Z_GREET("world") rc=', sy-subrc.
  WRITE: / '   GREETING =', lv_greeting.

  WRITE: / '== done — logic ran in Go, ABAP saw ordinary FMs =='.
