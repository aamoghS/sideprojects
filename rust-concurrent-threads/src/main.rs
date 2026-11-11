use std::thread;
use std::time::Instant;

const LIMIT: u64 = 3_000_000;

fn is_prime(n: u64) -> bool {
    if n < 2 {
        return false;
    }
    let mut d = 2;
    while d * d <= n {
        if n.is_multiple_of(d) {
            return false;
        }
        d += 1;
    }
    true
}

fn count_sequential(limit: u64) -> usize {
    (0..limit).filter(|&n| is_prime(n)).count()
}

fn count_parallel(limit: u64, threads: u64) -> usize {
    thread::scope(|s| {
        let handles: Vec<_> = (0..threads)
            .map(|t| {
                s.spawn(move || {
                    (t..limit)
                        .step_by(threads as usize)
                        .filter(|&n| is_prime(n))
                        .count()
                })
            })
            .collect();
        handles.into_iter().map(|h| h.join().unwrap()).sum()
    })
}

fn main() {
    let threads = thread::available_parallelism().map_or(4, |n| n.get() as u64);

    let start = Instant::now();
    let seq = count_sequential(LIMIT);
    let seq_time = start.elapsed();
    println!("1 thread:   {seq} primes below {LIMIT} in {seq_time:.2?}");

    let start = Instant::now();
    let par = count_parallel(LIMIT, threads);
    let par_time = start.elapsed();
    println!("{threads} threads: {par} primes below {LIMIT} in {par_time:.2?}");

    assert_eq!(seq, par, "both versions agree");
    println!(
        "speedup: {:.1}x",
        seq_time.as_secs_f64() / par_time.as_secs_f64()
    );
}
