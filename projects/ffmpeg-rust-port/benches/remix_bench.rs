use criterion::{black_box, criterion_group, criterion_main, Criterion};

// Simulate unoptimized mono to stereo
fn mono_to_stereo_unoptimized(input: &[f32]) -> Vec<f32> {
    let mut output = Vec::new();
    for &sample in input {
        output.push(sample);
        output.push(sample); // Duplicate to both channels
    }
    output
}

// Simulate optimized mono to stereo
fn mono_to_stereo_optimized(input: &[f32]) -> Vec<f32> {
    let mut output = Vec::with_capacity(input.len() * 2);
    for &sample in input {
        output.push(sample);
        output.push(sample); // Duplicate to both channels
    }
    output
}

fn criterion_benchmark(c: &mut Criterion) {
    let mut group = c.benchmark_group("remix_mono_to_stereo");

    // Test on a much larger vector to demonstrate allocation cost differences
    let input: Vec<f32> = (0..48000*10).map(|i| {
        let t = i as f32 / 48000.0;
        let freq = 440.0;
        (t * freq * 2.0 * std::f32::consts::PI).sin()
    }).collect();

    group.bench_function("mono_to_stereo_unoptimized", |b| b.iter(|| mono_to_stereo_unoptimized(black_box(&input))));
    group.bench_function("mono_to_stereo_optimized", |b| b.iter(|| mono_to_stereo_optimized(black_box(&input))));

    group.finish();
}

criterion_group!(benches, criterion_benchmark);
criterion_main!(benches);
