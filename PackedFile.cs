using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;
using System.Threading.Tasks;

namespace OptoPacker
{
    internal class PackedFile
    {
        public PackedFile()
        {
            FileCount = 0;
            UnpackedSize = 0;
            RootDirectory = new DirectoryEntry(".");
            Blobs = new List<BlobEntry>();
            Hash = Array.Empty<byte>();
        }

        public ulong FileCount { get; set; }
        public ulong UnpackedSize { get; set; }
        DirectoryEntry RootDirectory { get; set; }
        List<BlobEntry> Blobs { get; set; }
        byte[] Hash { get; set; }
        
        internal async Task AddFile(FileInfo file)
        {
            FileCount++;
            UnpackedSize += file.Size;
            await RootDirectory.AddFile(file);
        }
    }
}
