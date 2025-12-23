let add value =
  let open Yojson.Basic.Util in
  try
    let a = value |> member "a" |> to_int_option in
    let b = value |> member "b" |> to_int_option in
    match a, b with
    | Some a, Some b -> Ok (`Int (a + b))
    | _, _ -> Error "function is not valid"
  with Type_error (msg, _) -> Error ("input type error")
