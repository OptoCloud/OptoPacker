using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;
using System.Threading.Tasks;

namespace OptoPacker
{
    internal struct FileInfo
    {
        public string Path { get; set; }
        public UInt64 Size { get; set; }
        public byte[] Hash { get; set; }

        public FileInfo(string path, UInt64 size, byte[] hash)
        {
            Path = path;
            Size = size;
            Hash = hash;
        }
    }
}
