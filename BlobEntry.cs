using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;
using System.Threading.Tasks;

namespace OptoPacker
{
    internal class BlobEntry
    {
        public UInt32 Id { get; set; }
        public UInt64 Size { get; set; }
        public string Hash { get; set; }
    }
}
