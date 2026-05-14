use criterion::{black_box, criterion_group, criterion_main, Criterion};

// Simplified version of the function to benchmark since we can't easily import the private function
fn linear_resample_unoptimized(input: &[f32], ratio: f64) -> Vec<f32> {
    if input.is_empty() || ratio <= 0.0 {
        return Vec::new();
    }

    let output_len = (input.len() as f64 / ratio).ceil() as usize;
    let mut output = Vec::new();

    for i in 0..output_len {
        let input_pos = i as f64 / ratio;
        let input_idx = input_pos.floor() as usize;

        if input_idx + 1 < input.len() {
            let frac = input_pos - input_idx as f64;
            let sample = (input[input_idx] as f64 * (1.0 - frac) + input[input_idx + 1] as f64 * frac) as f32;
            output.push(sample.clamp(-1.0, 1.0));
        } else if input_idx < input.len() {
            output.push(input[input_idx]);
        }
    }

    output
}

fn linear_resample_optimized(input: &[f32], ratio: f64) -> Vec<f32> {
    if input.is_empty() || ratio <= 0.0 {
        return Vec::new();
    }

    let output_len = (input.len() as f64 / ratio).ceil() as usize;
    let mut output = Vec::with_capacity(output_len);

    for i in 0..output_len {
        let input_pos = i as f64 / ratio;
        let input_idx = input_pos.floor() as usize;

        if input_idx + 1 < input.len() {
            let frac = input_pos - input_idx as f64;
            let sample = (input[input_idx] as f64 * (1.0 - frac) + input[input_idx + 1] as f64 * frac) as f32;
            output.push(sample.clamp(-1.0, 1.0));
        } else if input_idx < input.len() {
            output.push(input[input_idx]);
        }
    }

    output
}

fn criterion_benchmark(c: &mut Criterion) {
    let mut group = c.benchmark_group("resample");
    // Generate 1 second of audio at 48kHz
    let input: Vec<f32> = (0..48000).map(|i| (i as f32 * 440.0 * 2.0 * std::f32::consts::PI / 48000.0).sin()).collect();

    // Test 48kHz to 44.1kHz downsampling
    let ratio = 48000.0 / 44100.0;

    group.bench_function("unoptimized", |b| b.iter(|| linear_resample_unoptimized(black_box(&input), black_box(ratio))));
    group.bench_function("optimized", |b| b.iter(|| linear_resample_optimized(black_box(&input), black_box(ratio))));

    // Test 44.1kHz to 48kHz upsampling
    let ratio2 = 44100.0 / 48000.0;
    group.bench_function("unoptimized_up", |b| b.iter(|| linear_resample_unoptimized(black_box(&input), black_box(ratio2))));
    group.bench_function("optimized_up", |b| b.iter(|| linear_resample_optimized(black_box(&input), black_box(ratio2))));

    group.finish();
}

criterion_group!(benches, criterion_benchmark);
criterion_main!(benches);
