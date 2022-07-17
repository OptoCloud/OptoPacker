using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;
using System.Threading.Tasks;

namespace OptoPacker
{
    internal class FileEntry
    {
        public string InputPath { get; set; }
        public string Name { get; set; }
        public UInt32 BlobId { get; set; }
    }
}
