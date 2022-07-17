namespace OptoPacker
{
    internal class DirectoryEntry
    {
        public DirectoryEntry(string name)
        {
            Name = name;
            Files = new List<FileEntry>();
            SubDirs = new Dictionary<string, DirectoryEntry>();
        }
        public string Name { get; set; }
        public List<FileEntry> Files { get; set; }
        public Dictionary<string, DirectoryEntry> SubDirs { get; set; }
        
        internal async Task AddFile(FileInfo file)
        {
            if (file.Path.Contains('\\') || file.Path.Contains('/'))
            {
                var parts = file.Path.Split('\\', '/');
                var dir = parts[0];
                if (!SubDirs.TryGetValue(dir, out var entry))
                {
                    entry = new DirectoryEntry(dir);
                    SubDirs.Add(dir, entry);
                }

                file.Path = string.Join('\\', parts[1..]);

                await entry.AddFile(file);
            }
        }
    }
}
