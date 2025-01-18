using System.ComponentModel.DataAnnotations;
using System.ComponentModel.DataAnnotations.Schema;

namespace StreamDb.Models;

public class User : BaseEntity 
{
    [Column("email")]
    [Required]
    public string Email { get; set; }
    
    [Column("names")]
    [Required]
    public string Names { get; set; }
    
    public ICollection<Streams> Streams { get; set; }
    public ICollection<Comments> Comments { get; set; }
}