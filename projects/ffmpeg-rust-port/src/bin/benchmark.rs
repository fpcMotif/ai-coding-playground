use std::time::Instant;
use std::hint::black_box;

fn main() {
    let num_samples = 44100 * 2; // 1 second of stereo audio
    let iterations = 10_000;

    let start = Instant::now();
    for _ in 0..iterations {
        let mut samples = Vec::new();
        for _ in 0..num_samples {
            samples.push(0.0);
        }
        black_box(samples);
    }
    let duration_push = start.elapsed();
    println!("Push loop duration: {:?}", duration_push);

    let start2 = Instant::now();
    for _ in 0..iterations {
        let samples = vec![0.0; num_samples];
        black_box(samples);
    }
    let duration_vec = start2.elapsed();
    println!("Vec macro duration: {:?}", duration_vec);

    println!("Improvement: {:.2}x", duration_push.as_secs_f64() / duration_vec.as_secs_f64());
}
