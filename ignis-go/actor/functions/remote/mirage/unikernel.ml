open Lwt.Infix
open Websocket
open Cohttp_mirage
open Cmdliner
open Handlers

let connect_id =
  let doc = Arg.info ~doc:"Connection ID of current executor." [ "id" ] in
  Arg.(required & opt (some string) None doc)

let uri =
  let doc = Arg.info ~doc:"The WS uri to connect." [ "uri" ] in
  Arg.(required & opt (some string) None doc)

let fail_if eq f = if eq then f () else Lwt.return_unit

module Mirage_websocket
  (Res: Resolver_mirage.S)
  (Conn : Conduit_mirage.S)
= struct
  module Channel = Mirage_channel.Make (Conn.Flow)
  module Input_channel = Input_channel.Make(Channel)
  module IO = Cohttp_mirage.IO (Channel)
  module WS = Websocket.Make (IO)
  module Endpoint = Conduit_mirage.Endpoint

  exception Connection_error of string
  let connection_error msg = Lwt.fail @@ Connection_error msg
  let protocol_error msg = Lwt.fail @@ Protocol_error msg

  let upgrade_headers headers key =
    Cohttp.Header.add_list headers [
      "Upgrade", "websocket";
      "Connection", "Upgrade";
      "Sec-Websocket-Key", key;
      "Sec-Websocket-Version", "13";
    ]

  let do_handshake headers url ic oc key =
    let headers = upgrade_headers headers key in
    let req = Cohttp.Request.make ~headers url in
    WS.Request.write ~flush:true (fun _w -> Lwt.return_unit) req oc
    >>= fun () ->
      (WS.Response.read ic >>= function
      | `Ok r -> Lwt.return r
      | `Eof -> connection_error "Connection closed during handshake"
      | `Invalid s -> protocol_error @@ "Invalid handshake: " ^ s)
    >>= fun response ->
      let open Cohttp.Code in
      let status = Cohttp.Response.status response in
      Logs.info (fun m ->
        m "Handshake response: %s%!" @@ string_of_status status);
      fail_if
        (is_error @@ code_of_status status)
        (fun () -> connection_error @@ string_of_status status)

  let do_connect resolver con uri =
    Res.resolve_uri ~uri resolver
    >>= fun endp ->
      Logs.info (fun m -> m "Connecting to %s" @@ Uri.to_string uri);
      Endpoint.client endp
    >>= fun client -> Conn.connect con client
    >|= fun flow -> 
      let oc = Channel.create flow in
      flow, Input_channel.create oc, oc

  type conn = {
    read_frame : unit -> Frame.t Lwt.t;
    write_frame : Frame.t -> unit Lwt.t;
    ch : IO.oc;
    close: unit -> unit Lwt.t;
  }

  let read conn = conn.read_frame ()
  let write conn frame = conn.write_frame frame
  let close conn = conn.close

  let connect_ws resolver con uri headers key =
    let uri = Uri.of_string uri in
    do_connect resolver con uri
    >>= fun (_flow, ic, oc) ->
      Lwt.catch
        (fun () -> do_handshake headers uri ic oc key)
        (fun exn -> Lwt.fail exn)
    >|= fun () ->
      Logs.info (fun m -> m "Connected to %s" @@ Uri.to_string uri);
      ic, oc

  let connect ?(rng = Websocket.Rng.init ()) resolver con uri headers =
    let open WS in
    let key = Base64.encode_exn (rng 16) in
    connect_ws resolver con uri headers key
    >|= fun (ic, oc) ->
      let read_frame = make_read_frame ?buf:None ~mode:(Client rng) ic oc in
      let read_frame () = Lwt.catch read_frame Lwt.fail in
      let buf = Buffer.create 128 in
      let write_frame frame =
        Logs.info (fun m -> m "Writing frame: %s" @@ Frame.show frame);
        Buffer.clear buf;
        Lwt.wrap2 (write_frame_to_buf ~mode:(Client rng)) buf frame
        >>= fun() ->
          let buf = Buffer.contents buf in
          Lwt.catch
            (fun () -> IO.write oc buf >>= fun () -> IO.flush oc)
            Lwt.fail
      in
      let close () = Channel.close oc >>= fun r -> r |> function
      | Ok () -> Lwt.return_unit
      | Error _ -> connection_error "Error closing channel" in
      let ch = oc in
      {read_frame; write_frame; ch; close}

  let make_text_frame content =
    Frame.create ~opcode:Frame.Opcode.Text ~content ()

  let make_pong_frame content =
    Frame.create ~opcode:Frame.Opcode.Pong ~content ()

  let start conn handler =
    let handler frame =
      let open Frame in
      Logs.info (fun m -> m "Received frame: %s" @@ show frame);
      match frame.opcode with
      | Opcode.Close -> connection_error "connection closed"
      | Opcode.Text -> handler conn frame.content
      | Opcode.Ping -> make_pong_frame frame.content |> write conn
      | _ -> Logs.info (fun m -> m "Received unknown frame"); Lwt.return_unit 
    in
    let rec loop () = conn.read_frame () >>= handler >>= loop in
    Lwt.catch loop (function
    | Connection_error ("connection closed") | End_of_file -> 
      Logs.info (fun m -> m "connection closed");
      Lwt.return_unit
    | exn ->
      Logs.err (fun m -> m "Error while reading: %s" @@ Printexc.to_string exn);
      conn.close () |> Lwt.ignore_result;
      Lwt.fail exn
    )
end

module Message = struct
  type it = {
    corr_id: string;
    topic: string;
    value: Yojson.Basic.t;
  }

  let topic msg = msg.topic

  let value msg = msg.value

  let corr_id msg = msg.corr_id

  type ot = {
    corr_id: string;
    topic: string;
    value: Yojson.Basic.t option;
    error: string option
  }

  let create_error corr_id topic error =
    { corr_id; topic; value = None; error = Some error }

  let create_value corr_id topic value =
    { corr_id; topic; value = Some value; error = None }

  let of_string msg = try
    let open Yojson.Basic.Util in
    let json = Yojson.Basic.from_string msg in
    let corr_id = json |> member "corr_id" |> to_string in
    let topic = json |> member "topic" |> to_string in
    let value = json |> member "value" in
    Some { corr_id; topic; value}
  with exn ->
    Logs.warn (fun m -> m "Recevied invalid message: %s %s"
      msg @@ Printexc.to_string exn);
    None

  let to_string msg =
    let value = match msg.value with
    | Some value -> value
    | None -> `Null in
    let error = match msg.error with
    | Some error -> `String error
    | None -> `Null in
    `Assoc [
      "corr_id", `String msg.corr_id;
      "topic", `String msg.topic;
      "value", value;
      "error", error;
    ] |> Yojson.Basic.to_string
end

module Store = Map.Make (String)

module Handler = struct
  type t = {
    name: string;
    applier: Yojson.Basic.t -> (Yojson.Basic.t, string) result;
  }

  let apply s msg = 
    let r = match Store.find_opt (Message.topic msg) s with
    | Some handler -> handler.applier msg.value 
    | None -> Error "no such handler" in
    match r with
    | Ok value -> Message.create_value msg.corr_id msg.topic value
    | Error error -> Message.create_error msg.corr_id msg.topic error

  let create name applier = {name; applier}
end

module Client
  (Res: Resolver_mirage.S)
  (Conn : Conduit_mirage.S)
= struct
  module WS = Mirage_websocket (Res) (Conn)  

  let on_start id conn =
    Logs.info (fun m -> m "Starting %s..." @@ id)

  let handle_msg store conn msg =
    let send = Handler.apply store msg in
    let text = Message.to_string send |> WS.make_text_frame in
    WS.write conn text

  let handle store conn text =
    match Message.of_string text with
    | Some msg -> handle_msg store conn msg
    | None -> Lwt.return_unit

  let start resolver con id uri =
    let headers = Cohttp.Header.of_list [
      "executor_id", id
    ] in
    let open Handlers in
    let functions = List.map (fun (a, b) -> a, (Handler.create a b)) handlers in
    let store = Store.of_list functions in
    WS.connect resolver con uri headers
    >>= fun conn ->
      on_start id conn;
      WS.start conn (handle store)
end
