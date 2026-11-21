#pragma once

#include <cstddef>
#include <cstdint>

namespace attendance {
namespace timeutil {

// Starts SNTP once after Wi-Fi is available. Never blocks.
void ensureStarted(bool wifiConnected);

// True once the clock is past 2023-11-14 UTC.
bool clockValid();

// Seconds since Unix epoch, or 0 when the clock is not synchronized.
int64_t nowEpoch();

// Formats a UTC ISO-8601 timestamp. Returns false for invalid/zero epochs or
// if the output buffer is too small.
bool formatIso8601(int64_t epochSeconds, char* out, size_t outLength);

}  // namespace timeutil
}  // namespace attendance
