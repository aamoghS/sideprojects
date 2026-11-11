use std::sync::{Arc, Mutex};
use std::thread;

const THREADS: usize = 8;
const INCREMENTS: usize = 100_000;

fn main() {
    let count = Arc::new(Mutex::new(0usize));

    let handles: Vec<_> = (0..THREADS)
        .map(|_| {
            let count = Arc::clone(&count);
            thread::spawn(move || {
                for _ in 0..INCREMENTS {
                    // The guard unlocks the mutex when it goes out of scope.
                    *count.lock().unwrap() += 1;
                }
            })
        })
        .collect();

    println!(
        "Arc strong count while threads run: {}",
        Arc::strong_count(&count)
    );
    for handle in handles {
        handle.join().unwrap();
    }

    let total = *count.lock().unwrap();
    println!("expected {}, got {total}", THREADS * INCREMENTS);
    assert_eq!(total, THREADS * INCREMENTS, "no increments lost");
}
