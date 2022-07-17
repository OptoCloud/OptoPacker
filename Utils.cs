using System;
using System.Collections.Generic;
using System.Linq;
using System.Security.Cryptography;
using System.Text;
using System.Threading.Tasks;

namespace OptoPacker
{
    internal static class Utils
    {
        [ThreadStatic]
        static Random? _random;
        static Random Random => _random ??= new Random((int)DateTime.Now.Ticks);

        [ThreadStatic]
        static SHA256? _sha256;
        static SHA256 Sha256 => _sha256 ??= SHA256.Create();
        
        public static async Task<(byte[] hash, long length)> HashAsync(Stream stream)
        {
            var hash = await Sha256.ComputeHashAsync(stream);
            return (hash, (int)stream.Length);
        }
        
        public static async Task<(byte[] hash, long length)> HashAsync(string path)
        {
            using var stream = File.OpenRead(path);
            return await HashAsync(stream);
        }

        public static async IAsyncEnumerable<FileInfo> HashAllAsync(IEnumerable<string> files)
        {
            foreach (var file in files)
            {
                byte[] hash;
                long size;
                try
                {
                    (hash, size) = await Utils.HashAsync(file);
                }
                catch (Exception ex)
                {
                    Console.WriteLine($"Skipping {Path.GetFileName(file)}: {ex.Message}");
                    continue;
                }
                if (hash == null || size <= 0)
                {
                    Console.WriteLine($"Skipping {Path.GetFileName(file)}: Invalid hash or size");
                    continue;
                }

                yield return new FileInfo(file, (ulong)size, hash);
            }
        }

        public static string FormatNumberByteSize(ulong bytes)
        {
            uint i = 0;
            float f = bytes;
            while (f > 1024f)
            {
                f /= 1024f;
                i++;
            }

            string unit = i switch
            {
                0 => " B",
                1 => "KB",
                2 => "MB",
                3 => "GB",
                4 => "TB",
                5 => "PB",
                6 => "EB",
                7 => "ZB",
                8 => "YB",
                _ => "??"
            };

            return $"{f:0.00} {unit}".PadLeft(10);
        }
    }
}
