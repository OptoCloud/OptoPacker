using System.Text.RegularExpressions;

namespace OptoPacker
{
    internal static class GitignoreParser
    {
        static bool IsSystemPath(string path) => 
            path.Count(c => c == '\\') <= 1 && (path.EndsWith("\\$RECYCLE.BIN") || path.EndsWith("\\System Volume Information"));
        
        static string GetRelativeName(string path, string relativeTo, bool isDir)
        {
            var relativePath = path[relativeTo.Length..];
            if (relativePath.StartsWith('\\'))
                relativePath = relativePath[1..];
            if (isDir && !relativePath.EndsWith('\\'))
                relativePath += '\\';
            return relativePath;
        }

        static IEnumerable<string> Crawl(string path, Dictionary<string, GitRegex[]> allPatterns, IEnumerable<GitRegex> activePatterns)
        {
            if (IsSystemPath(path)) yield break;
            
            var results = new List<string>();
            var localActivePatterns = new List<GitRegex>(activePatterns);

            // If we have a pattern for current dir, use it
            if (allPatterns.TryGetValue(path, out var patterns))
                localActivePatterns.AddRange(patterns);

            string[] files;
            try
            {
                files = Directory.GetFiles(path, "*", SearchOption.TopDirectoryOnly);
            }
            catch (Exception)
            {
                yield break;
            }

            foreach (var file in files)
            {
                if (localActivePatterns.All(p => !p.Regex.IsMatch(GetRelativeName(file, p.Path, false)))) yield return file;
            }
            foreach (var entry in Directory.GetDirectories(path, "*", SearchOption.TopDirectoryOnly))
            {
                if (entry.EndsWith(".git")) continue;
                if (localActivePatterns.All(p => !p.Regex.IsMatch(GetRelativeName(entry, p.Path, true))))
                {
                    foreach (var file in Crawl(entry, allPatterns, localActivePatterns)) yield return file;
                }
            }
        }
        static IEnumerable<string> GetFiles(string path, string pattern, bool topDir = true)
        {
            if (IsSystemPath(path)) yield break;

            string[] array;
            try
            {
                array = Directory.GetFiles(path, pattern, SearchOption.AllDirectories);
            }
            catch (Exception)
            {
                if (!topDir) yield break;
                array = Array.Empty<string>();
            }

            if (array.Length > 0)
            {
                foreach (var file in array)
                {
                    yield return file;
                }
                yield break;
            }
            
            try
            {
                array = Directory.GetDirectories(path);
            }
            catch (Exception)
            {
                array = Array.Empty<string>();
            }
            
            foreach (var subPath in array)
            {
                foreach (var item in GetFiles(subPath, pattern, false))
                {
                    yield return item;
                }
            }
        }
        public static IEnumerable<string> GetTrackedFiles(string basePath)
        {
            var IgnorePatterns = GetFiles(basePath, "*.gitignore")
                .ToDictionary(
                    (p) => Path.GetDirectoryName(p)!,
                    (p) => ParseGitignore(p).ToArray()
                );
            if (IgnorePatterns == null) return null;

            // Get tracked files
            var patternStack = new List<GitRegex>();

            // Crawl
            return Crawl(basePath, IgnorePatterns!, new GitRegex[0]);
        }

        static string UnescapeString(string str)
        {
            if (string.IsNullOrEmpty(str)) return string.Empty;

            int spaces = 0;
            string result = string.Empty;

            for (int i = 0; i < str.Length;)
            {
                var c = str[i++];
                switch (c)
                {
                    case '\\':
                        if (i < str.Length)
                        {
                            result += str[i++] switch { ' ' => ' ', '#' => '#', _ => throw new Exception("Wtf") };
                        }
                        break;
                    case '#':
                        return result;
                    case ' ':
                        spaces++;
                        break;
                    default:
                        if (spaces != 0)
                        {
                            result += new string(' ', spaces);
                            spaces = 0;
                        }
                        result += c;
                        break;
                }
            }

            return result;
        }

        struct GitRegex
        {
            public string Path;
            public Regex Regex;
            public bool IsNegative;
        }
        static IEnumerable<GitRegex> ParseGitignore(string gitignorePath)
        {
            // Read file
            var lines = File.ReadAllLines(gitignorePath);

            // Convert ignore patterns to regex
            foreach (var line in lines)
            {
                var pattern = UnescapeString(line);

                if (string.IsNullOrEmpty(pattern)) continue;

                bool isNegative = pattern[0] == '!';
                if (isNegative)
                    pattern = pattern[1..];

                bool anyDepth = pattern.IndexOf('/') == pattern.Length - 1 || !pattern.Contains('/');

                if (pattern[0] == '/' || pattern[0] == '\\')
                    pattern = pattern[1..];

                pattern = $"^{Regex.Escape(pattern)}$"
                    .Replace(/* lang=regex */ @"/\*\*/", /* lang=regex */ "/.*/")
                    .Replace(/* lang=regex */ @"^\*\*/", /* lang=regex */  ".*/")
                    .Replace(/* lang=regex */ @"/\*\*$", /* lang=regex */ "/.*")
                    .Replace(/* lang=regex */ @"\*\*",   /* lang=regex */ "/^*")
                    .Replace(@"\[", "[")
                    .Replace(@"\]", "]")
                    .Replace(@"\?", "/^")
                    .Replace(@"\*", "/^*");

                if (anyDepth && !pattern.StartsWith(/*lang = regex */ ".*/"))
                    pattern = ".*/" + pattern[1..];
                if (!pattern.EndsWith("/$") && !pattern.EndsWith(".*"))
                    pattern = pattern[..^1] + "/?$";

                pattern = pattern.Replace("/", /* lang=regex */ @"[/\\]").Replace(@"[/\\]^", /* lang=regex */ @"[^/\\]");

                yield return new GitRegex()
                {
                    Path = Path.GetDirectoryName(gitignorePath)!,
                    Regex = new Regex(pattern),
                    IsNegative = pattern.StartsWith("!")
                };
            }
        }
    }
}
