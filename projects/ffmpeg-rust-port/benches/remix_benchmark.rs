use criterion::{black_box, criterion_group, criterion_main, Criterion};

// Baseline implementations
fn stereo_to_mono_baseline(input: &[f32]) -> Vec<f32> {
    let mut output = Vec::new();
    for i in (0..input.len()).step_by(2) {
        if i + 1 < input.len() {
            let avg = (input[i] + input[i + 1]) / 2.0;
            output.push(avg);
        }
    }
    output
}

fn mono_to_stereo_baseline(input: &[f32]) -> Vec<f32> {
    let mut output = Vec::new();
    for &sample in input {
        output.push(sample);
        output.push(sample); // Duplicate to both channels
    }
    output
}

fn stereo_left_baseline(input: &[f32]) -> Vec<f32> {
    let mut output = Vec::new();
    for i in (0..input.len()).step_by(2) {
        output.push(input[i]);
    }
    output
}

fn stereo_right_baseline(input: &[f32]) -> Vec<f32> {
    let mut output = Vec::new();
    for i in (0..input.len()).step_by(2) {
        if i + 1 < input.len() {
            output.push(input[i + 1]);
        }
    }
    output
}

// Optimized implementations
fn stereo_to_mono_optimized(input: &[f32]) -> Vec<f32> {
    input.chunks_exact(2).map(|chunk| (chunk[0] + chunk[1]) / 2.0).collect()
}

fn mono_to_stereo_optimized(input: &[f32]) -> Vec<f32> {
    let mut output = Vec::with_capacity(input.len() * 2);
    for &sample in input {
        output.push(sample);
        output.push(sample);
    }
    output
}

fn stereo_left_optimized(input: &[f32]) -> Vec<f32> {
    input.chunks_exact(2).map(|chunk| chunk[0]).collect()
}

fn stereo_right_optimized(input: &[f32]) -> Vec<f32> {
    input.chunks_exact(2).map(|chunk| chunk[1]).collect()
}

fn benchmark_remix(c: &mut Criterion) {
    let sample_count = 192_000;
    let input_samples: Vec<f32> = (0..sample_count).map(|i| (i as f32).sin()).collect();

    let mut group1 = c.benchmark_group("Remix Stereo to Mono");
    group1.bench_function("baseline", |b| b.iter(|| stereo_to_mono_baseline(black_box(&input_samples))));
    group1.bench_function("optimized", |b| b.iter(|| stereo_to_mono_optimized(black_box(&input_samples))));
    group1.finish();

    let mut group2 = c.benchmark_group("Remix Mono to Stereo");
    group2.bench_function("baseline", |b| b.iter(|| mono_to_stereo_baseline(black_box(&input_samples))));
    group2.bench_function("optimized", |b| b.iter(|| mono_to_stereo_optimized(black_box(&input_samples))));
    group2.finish();

    let mut group3 = c.benchmark_group("Remix Stereo Left");
    group3.bench_function("baseline", |b| b.iter(|| stereo_left_baseline(black_box(&input_samples))));
    group3.bench_function("optimized", |b| b.iter(|| stereo_left_optimized(black_box(&input_samples))));
    group3.finish();

    let mut group4 = c.benchmark_group("Remix Stereo Right");
    group4.bench_function("baseline", |b| b.iter(|| stereo_right_baseline(black_box(&input_samples))));
    group4.bench_function("optimized", |b| b.iter(|| stereo_right_optimized(black_box(&input_samples))));
    group4.finish();
}

criterion_group!(benches, benchmark_remix);
criterion_main!(benches);
