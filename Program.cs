Console.WriteLine("Gathering files... (This might take a couple minutes)");

var trackedFiles = OptoPacker.GitignoreParser.GetTrackedFiles(@"H:\");

//File.WriteAllLines(@"H:\trackedFiles.txt", trackedFiles);

Console.WriteLine("Indexing files...");
OptoPacker.PackedFile packedFile = new OptoPacker.PackedFile();

await foreach (var file in OptoPacker.Utils.HashAllAsync(trackedFiles))
{
    Console.WriteLine($"Adding: {BitConverter.ToString(file.Hash).Replace("-", "")} - {OptoPacker.Utils.FormatNumberByteSize(file.Size)} - {file}");

    await packedFile.AddFile(file);
}