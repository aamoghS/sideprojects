use std::thread;
use std::time::Duration;

fn main() {
    let handles: Vec<thread::JoinHandle<u32>> = (1..=3)
        .map(|id| {
            thread::spawn(move || {
                for step in 1..=3 {
                    println!("worker {id}: step {step}");
                    thread::sleep(Duration::from_millis(10 * u64::from(id)));
                }
                id * 100
            })
        })
        .collect();

    println!("main: spawned {} workers, now waiting", handles.len());

    for handle in handles {
        let result = handle.join().expect("worker panicked");
        println!("main: worker returned {result}");
    }
}
