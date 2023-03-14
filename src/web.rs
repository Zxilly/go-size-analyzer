use actix_web::{get, Result, HttpRequest, HttpResponse, Responder, HttpServer, App, middleware};
use actix_web::http::StatusCode;
use actix_web::http::header::ContentType;
use crate::artifact::Packages;
use crate::utils::run_blocking;

#[get("/")]
async fn index() -> Result<impl Responder> {
    Ok(HttpResponse::build(StatusCode::OK)
        .content_type(ContentType::html())
        .body(include_str!("./web/dist/index.html")))
}

#[get("/csv")]
async fn csv(req: HttpRequest) -> Result<impl Responder> {
    let csv_str = req.app_data::<String>().unwrap();
    Ok(HttpResponse::build(StatusCode::OK)
        .content_type(ContentType::plaintext())
        .body(csv_str.clone()))
}

pub(crate) fn start(port: u16, packages: Packages) {
    let csv_str = packages.into_csv();

    println!("Starting web server on: http://127.0.0.1:{}", port);
    println!("Press Ctrl+C to stop");

    run_blocking(
        HttpServer::new(move || {
            App::new()
                .wrap(middleware::Compress::default())
                .app_data(csv_str.clone())
                .service(index)
                .service(csv)
        }).bind(("127.0.0.1", port))
            .unwrap()
            .workers(2)
            .run()
    )
}
