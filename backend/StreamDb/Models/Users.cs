using System.ComponentModel.DataAnnotations;
using System.ComponentModel.DataAnnotations.Schema;

namespace StreamDb.Models;

public sealed class User : BaseEntity 
{
    [Required]
    [MaxLength(100)]
    public string FirstName { get; set; } = null!;

    [Required]
    [MaxLength(100)]
    public string LastName { get; set; } = null!;

    [Required]
    [EmailAddress]
    [MaxLength(255)]
    public string Email { get; set; } = null!;

    [Required]
    [MaxLength(1000)]
    public string ProfileImageUrl { get; set; } = null!;

    [Required]
    [MaxLength(255)]
    [Index(IsUnique = true)]
    public string ClerkId { get; init; } = null!;
}