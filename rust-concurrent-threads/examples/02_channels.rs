use std::sync::mpsc;
use std::thread;
use std::time::Duration;

fn main() {
    let (tx, rx) = mpsc::channel::<String>();

    for id in 1..=3 {
        let tx = tx.clone();
        thread::spawn(move || {
            for n in 1..=2 {
                tx.send(format!("producer {id} says #{n}")).unwrap();
                thread::sleep(Duration::from_millis(5));
            }
            println!("producer {id} done, dropping its sender");
        });
    }
    // Drop main's sender, otherwise the receive loop below never ends.
    drop(tx);

    for message in rx {
        println!("consumer got: {message}");
    }
    println!("all senders dropped, channel closed");
}
