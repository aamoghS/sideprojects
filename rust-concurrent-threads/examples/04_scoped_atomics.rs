use std::sync::atomic::{AtomicUsize, Ordering};
use std::thread;

fn main() {
    let words = [
        "thread", "scope", "borrow", "atomic", "mutex", "channel", "arc", "join",
    ];
    let long_words = AtomicUsize::new(0);
    let mut lengths = vec![0usize; words.len()];

    thread::scope(|s| {
        // Each thread gets a disjoint &mut slice of `lengths`, so no lock is needed.
        for (chunk_id, (word_chunk, len_chunk)) in
            words.chunks(3).zip(lengths.chunks_mut(3)).enumerate()
        {
            let long_words = &long_words;
            s.spawn(move || {
                for (word, len) in word_chunk.iter().zip(len_chunk.iter_mut()) {
                    *len = word.len();
                    if word.len() > 5 {
                        long_words.fetch_add(1, Ordering::Relaxed);
                    }
                }
                println!("chunk {chunk_id} handled {word_chunk:?}");
            });
        }
    });

    // Scope has ended: every thread is joined, the borrows are over.
    for (word, len) in words.iter().zip(&lengths) {
        println!("{word:>8} = {len}");
    }
    println!(
        "words longer than 5 letters: {}",
        long_words.load(Ordering::Relaxed)
    );
}
