using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;
using System.Threading.Tasks;

namespace OptoPacker
{
    internal static class Packer
    {
        public static void Pack(IEnumerable<string> files)
        {
            PackedFile packed = new PackedFile();
            foreach (string file in files)
            {
                //packed.AddFile(file);
            }
        }
    }
}
